// Package pipeline turns flow records into stored rows: it enriches them,
// normalises byte counts to the link layer, spreads long flows over the
// minutes they covered, marks duplicate reports and maintains rollups.
package pipeline

import (
	"context"
	"fmt"
	"hash/maphash"
	"log"
	"math"
	"net/netip"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/store"
)

// Config tunes the pipeline.
type Config struct {
	// L2Overhead is added per packet to IP-layer byte counts to estimate
	// link-layer octets (Ethernet header + FCS = 18).
	L2Overhead int
}

// Pipeline is safe for concurrent Submit calls.
type Pipeline struct {
	dmu    sync.Mutex
	delays delayHist // how late data arrives
	cfg    Config
	st     *store.Store
	Inv    *enrich.Inventory
	ASN    *enrich.ASNDB
	Thr    *enrich.Threats

	in  chan []flow.Record
	ctr chan []flow.IfCounters

	mu     sync.Mutex
	batch  []store.Row
	rTS    map[store.TSKey]*store.Counters
	rHost  map[store.HostKey]*store.Counters
	rDim   map[store.DimKey]*store.Counters
	dedup  map[time.Time]map[uint64]uint32 // minute → 5-tuple hash → origin hash
	newest time.Time                       // newest minute in dedup
	seed   maphash.Seed

	Records atomic.Uint64
	Rows    atomic.Uint64
	DupRows atomic.Uint64
	// DedupFull counts flows not checked for duplicates because the minute
	// already held dedupMaxPerMinute flows.
	DedupFull atomic.Uint64
	Dropped   atomic.Uint64
	WriteErrs atomic.Uint64
	lastErr   atomic.Value
}

type dkey struct {
	src, dst     netip.Addr
	sport, dport uint16
	proto        uint8
}

type origin struct {
	exporter netip.Addr
	domain   uint32
	obs      uint32
	inIf     uint32
	outIf    uint32
	dir      uint8
}

func New(cfg Config, st *store.Store, inv *enrich.Inventory, asn *enrich.ASNDB, thr *enrich.Threats) *Pipeline {
	if cfg.L2Overhead <= 0 {
		cfg.L2Overhead = 18
	}
	return &Pipeline{
		cfg: cfg, st: st, Inv: inv, ASN: asn, Thr: thr,
		in:    make(chan []flow.Record, 4096),
		ctr:   make(chan []flow.IfCounters, 256),
		rTS:   map[store.TSKey]*store.Counters{},
		rHost: map[store.HostKey]*store.Counters{},
		rDim:  map[store.DimKey]*store.Counters{},
		dedup: map[time.Time]map[uint64]uint32{},
		seed:  maphash.MakeSeed(),
	}
}

// Submit queues records; when the queue is full they are counted as dropped.
func (p *Pipeline) Submit(recs []flow.Record) {
	if len(recs) == 0 {
		return
	}
	select {
	case p.in <- recs:
	default:
		p.Dropped.Add(uint64(len(recs)))
	}
}

func (p *Pipeline) SubmitCounters(c []flow.IfCounters) {
	if len(c) == 0 {
		return
	}
	select {
	case p.ctr <- c:
	default:
	}
}

// Ingest processes records synchronously (used for backfill).
func (p *Pipeline) Ingest(recs []flow.Record) {
	p.mu.Lock()
	for i := range recs {
		p.process(&recs[i])
	}
	p.mu.Unlock()
}

// IngestCounters stores counter snapshots synchronously (used for backfill).
func (p *Pipeline) IngestCounters(cs []flow.IfCounters) {
	for _, c := range cs {
		if err := p.st.AppendCounter(c.Time, c.Exporter.Unmap().String(), c.IfIndex, c.Speed, c.InOctets, c.OutOctets, c.InPkts, c.OutPkts); err != nil {
			p.fail(err)
		}
	}
}

// Run processes queued records until ctx ends, flushing rows every second,
// rollups every minute and sealing hourly.
func (p *Pipeline) Run(ctx context.Context) {
	rowTick := time.NewTicker(time.Second)
	rollTick := time.NewTicker(time.Minute)
	sealTick := time.NewTicker(5 * time.Minute)
	defer rowTick.Stop()
	defer rollTick.Stop()
	defer sealTick.Stop()
	for {
		select {
		case <-ctx.Done():
			p.FlushRows()
			p.FlushRollups()
			return
		case recs := <-p.in:
			p.noteDelay(recs, time.Now())
			p.mu.Lock()
			for i := range recs {
				p.process(&recs[i])
			}
			big := len(p.batch) > 50000
			p.mu.Unlock()
			if big {
				p.FlushRows()
			}
		case cs := <-p.ctr:
			for _, c := range cs {
				if err := p.st.AppendCounter(c.Time, c.Exporter.Unmap().String(), c.IfIndex, c.Speed, c.InOctets, c.OutOctets, c.InPkts, c.OutPkts); err != nil {
					p.fail(err)
				}
			}
		case <-rowTick.C:
			p.FlushRows()
		case now := <-rollTick.C:
			p.FlushRollups()
			p.updateSettle(now)
		case now := <-sealTick.C:
			p.FlushRows()
			if err := p.st.Seal(now); err != nil {
				p.fail(err)
			}
			if err := p.st.Retain(now); err != nil {
				p.fail(err)
			}
		}
	}
}

func (p *Pipeline) fail(err error) {
	p.WriteErrs.Add(1)
	p.lastErr.Store(err.Error())
	log.Printf("pipeline: %v", err)
}

// LastError returns the most recent write error text, if any.
func (p *Pipeline) LastError() string {
	if v, ok := p.lastErr.Load().(string); ok {
		return v
	}
	return ""
}

// FlushRows writes pending rows.
func (p *Pipeline) FlushRows() {
	p.mu.Lock()
	b := p.batch
	p.batch = nil
	p.mu.Unlock()
	if len(b) == 0 {
		return
	}
	if err := p.st.AppendRows(b); err != nil {
		p.fail(err)
		return
	}
	p.Rows.Add(uint64(len(b)))
}

// FlushRollups writes accumulated rollup rows.
func (p *Pipeline) FlushRollups() {
	p.mu.Lock()
	ts, host, dim := p.rTS, p.rHost, p.rDim
	p.rTS, p.rHost, p.rDim = map[store.TSKey]*store.Counters{}, map[store.HostKey]*store.Counters{}, map[store.DimKey]*store.Counters{}
	p.mu.Unlock()
	if len(ts)+len(host)+len(dim) == 0 {
		return
	}
	if err := p.st.AppendRollups(ts, host, dim); err != nil {
		p.fail(err)
	}
}

func ipv4num(a netip.Addr) uint32 {
	if !a.Is4() {
		return 0
	}
	b := a.As4()
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// process converts one record to rows; p.mu must be held.
func (p *Pipeline) process(r *flow.Record) {
	p.Records.Add(1)
	src, dst := r.Src.Unmap(), r.Dst.Unmap()
	if !src.IsValid() || !dst.IsValid() {
		return
	}
	mult := r.Mult
	if mult <= 0 {
		mult = 1
	}
	srcIsClient := enrich.Role(r)
	client, server := src, dst
	cport, sport := r.SrcPort, r.DstPort
	if !srcIsClient {
		client, server = dst, src
		cport, sport = r.DstPort, r.SrcPort
	}
	_ = cport
	cliInt, srvInt := p.Inv.Internal(client), p.Inv.Internal(server)
	var dir uint8
	switch {
	case cliInt && !srvInt:
		dir = store.DirOutbound
	case !cliInt && srvInt:
		dir = store.DirInbound
	case cliInt && srvInt:
		dir = store.DirInternal
	default:
		dir = store.DirTransit
	}
	var peer netip.Addr
	switch dir {
	case store.DirOutbound, store.DirTransit:
		peer = server
	case store.DirInbound:
		peer = client
	}
	var cc, org string
	var asn uint32
	if peer.IsValid() {
		asn, cc, org = p.ASN.Lookup(peer)
		if asn == 0 {
			if peer == dst {
				asn = r.DstAS
			} else {
				asn = r.SrcAS
			}
		}
	}
	segIP := client
	if dir == store.DirInbound {
		segIP = server
	}
	threat := p.Thr.Match(server)
	if threat == "" {
		threat = p.Thr.Match(client)
	}
	svcPort := sport
	if r.Proto != 6 && r.Proto != 17 && r.Proto != 132 {
		svcPort = 0
	}
	app := enrich.App(r.Proto, svcPort)

	pkts := float64(r.Packets) * mult
	var wire, l3 float64
	if r.L2 {
		wire = float64(r.Bytes) * mult
		l3 = math.Max(0, wire-pkts*18)
	} else {
		l3 = float64(r.Bytes) * mult
		oh := float64(p.cfg.L2Overhead)
		if r.VLAN != 0 {
			oh += 4
		}
		wire = l3 + pkts*oh
	}

	base := store.Row{
		Src: src.String(), Dst: dst.String(), SPort: r.SrcPort, DPort: r.DstPort, Proto: r.Proto,
		Client: client.String(), Server: server.String(), Cli4: ipv4num(client), Srv4: ipv4num(server),
		SvcPort: svcPort, App: app, Exporter: r.Exporter.Unmap().String(), SrcType: uint8(r.Source),
		InIf: r.InIf, OutIf: r.OutIf, VLAN: r.VLAN, Encap: uint8(r.Encap), Dir: dir,
		PeerCC: cc, PeerASN: asn, PeerOrg: org, Segment: p.Inv.Segment(segIP), Threat: threat,
		TCPFlags: r.TCPFlags, Mult: float32(mult), SampKnown: r.SamplingKnown, CliInt: cliInt, SrvInt: srvInt,
	}
	key := dkey{src, dst, r.SrcPort, r.DstPort, r.Proto}
	org2 := origin{r.Exporter, r.Domain, r.Observation, r.InIf, r.OutIf, r.Direction}
	obsIf := observedIf(r)

	slices := minuteSlices(r.Start, r.End)
	// Each minute gets the rounded running total minus what earlier minutes
	// got, so the shares are never negative and add up exactly. (Rounding
	// each share on its own could leave the last one below zero, which
	// wrapped to nearly 2^64 as an unsigned counter.)
	var cum float64
	var bDone, wDone, pDone uint64
	for i, sl := range slices {
		row := base
		row.TS = sl.t
		cum += sl.frac
		if i == len(slices)-1 {
			cum = 1
			row.Flows = 1
		}
		b, w, pk := share(l3, cum, bDone), share(wire, cum, wDone), share(pkts, cum, pDone)
		bDone, wDone, pDone = bDone+b, wDone+w, pDone+pk
		row.Bytes, row.Wire, row.Pkts = b, w, pk
		row.Dup = p.isDup(sl.t, key, org2)
		p.batch = append(p.batch, row)
		p.noteObs(&row, obsIf)
		if row.Dup {
			p.DupRows.Add(1)
			continue
		}
		p.rollup(&row, peer.IsValid())
	}
}

type slice struct {
	t    time.Time
	frac float64
}

// minuteSlices splits [start, end] into minute buckets weighted by overlap.
// share is how much of total belongs to this minute: the rounded running
// total up to it, less what the minutes before it already got.
func share(total, cum float64, done uint64) uint64 {
	upTo := math.Round(total * min(cum, 1))
	if upTo <= float64(done) {
		return 0
	}
	return uint64(upTo) - done
}

func minuteSlices(start, end time.Time) []slice {
	start, end = start.UTC(), end.UTC()
	if end.IsZero() {
		end = time.Now().UTC()
	}
	if start.IsZero() || !start.Before(end) {
		return []slice{{end.Truncate(time.Minute), 1}}
	}
	if end.Sub(start) > 6*time.Hour { // implausible; keep the last hour
		start = end.Add(-time.Hour)
	}
	total := end.Sub(start).Seconds()
	var out []slice
	for b := start.Truncate(time.Minute); b.Before(end); b = b.Add(time.Minute) {
		lo, hi := b, b.Add(time.Minute)
		if lo.Before(start) {
			lo = start
		}
		if hi.After(end) {
			hi = end
		}
		ov := hi.Sub(lo).Seconds()
		if ov <= 0 {
			continue
		}
		out = append(out, slice{b, ov / total})
	}
	if len(out) == 0 {
		return []slice{{end.Truncate(time.Minute), 1}}
	}
	return out
}

// dedupWindow is how far back reports of the same minute are compared:
// exporters report a flow up to their active timeout (plus any wait for a
// sampling rate) after it happened.
const dedupWindow = 8 * time.Minute

// dedupMaxPerMinute caps the flows remembered per minute (about 25 MB), so a
// flood of distinct flows cannot grow memory beyond 8 × that. Flows beyond
// it are counted in DedupFull and treated as not duplicated.
var dedupMaxPerMinute = 1_000_000

// isDup reports whether another observation point already reported this
// packet direction in this minute.
//
// Only hashes are kept: 64 bits of the 5-tuple and 32 of the observation
// point, about 25 bytes per flow and minute instead of about 200. At the
// design rate (300,000 flows a minute, nearly all with distinct ports) the
// table holds about 60 MB. Two different flows in the same minute share a
// 64-bit hash with a probability of about 1 in 10^9 per minute.
func (p *Pipeline) isDup(t time.Time, k dkey, o origin) bool {
	if t.After(p.newest) {
		p.newest = t
		for m := range p.dedup {
			if m.Before(t.Add(-dedupWindow)) {
				delete(p.dedup, m)
			}
		}
	}
	if t.Before(p.newest.Add(-dedupWindow)) {
		return false // older than anything kept: nothing to compare with
	}
	m := p.dedup[t]
	if m == nil {
		m = map[uint64]uint32{}
		p.dedup[t] = m
	}
	kh := maphash.Comparable(p.seed, k)
	oh := uint32(maphash.Comparable(p.seed, o))
	prev, ok := m[kh]
	if !ok {
		if len(m) >= dedupMaxPerMinute {
			if p.DedupFull.Add(1) == 1 {
				log.Printf("pipeline: more than %d distinct flows in a minute; flows beyond that are not checked for duplicates", dedupMaxPerMinute)
			}
			return false
		}
		m[kh] = oh
		return false
	}
	return prev != oh
}

// observedIf is the interface a flow was sampled on: the sFlow data source
// when it is an interface, else the ingress or egress interface as the
// exporter's flowDirection says; 0 when it cannot be told.
func observedIf(r *flow.Record) uint32 {
	switch {
	case r.Observation != 0 && r.Observation>>24 == 0:
		return r.Observation & 0xFFFFFF
	case r.Direction == 0:
		return r.InIf
	case r.Direction == 1:
		return r.OutIf
	}
	return 0
}

// noteObs counts each row, duplicates included, under the interface it was
// sampled on (dim obsif, "exporter/ifindex"), or under its exporter when that
// is unknown (dim obsunk). The interface list uses it to tell the sampled
// interfaces from the ones only seen as the other end of their flows.
func (p *Pipeline) noteObs(r *store.Row, obsIf uint32) {
	c := store.Counters{Bytes: r.Bytes, Wire: r.Wire, Pkts: r.Pkts, Flows: uint64(r.Flows)}
	k := store.DimKey{TS: r.TS.Truncate(time.Hour), Dim: "obsunk", Val: r.Exporter}
	if obsIf != 0 {
		k.Dim, k.Val = "obsif", r.Exporter+"/"+strconv.FormatUint(uint64(obsIf), 10)
	}
	if v := p.rDim[k]; v != nil {
		v.Add(c)
	} else {
		p.rDim[k] = &c
	}
}

func (p *Pipeline) rollup(r *store.Row, hasPeer bool) {
	c := store.Counters{Bytes: r.Bytes, Wire: r.Wire, Pkts: r.Pkts, Flows: uint64(r.Flows)}
	add := func(m map[store.DimKey]*store.Counters, k store.DimKey) {
		if v := m[k]; v != nil {
			v.Add(c)
		} else {
			cc := c
			m[k] = &cc
		}
	}
	tk := store.TSKey{TS: r.TS, App: r.App}
	if v := p.rTS[tk]; v != nil {
		v.Add(c)
	} else {
		cc := c
		p.rTS[tk] = &cc
	}
	hour := r.TS.Truncate(time.Hour)
	for _, hk := range []store.HostKey{{TS: hour, IP: r.Client, Role: 1, Internal: r.CliInt}, {TS: hour, IP: r.Server, Role: 2, Internal: r.SrvInt}} {
		if v := p.rHost[hk]; v != nil {
			v.Add(c)
		} else {
			cc := c
			p.rHost[hk] = &cc
		}
	}
	country := r.PeerCC
	if r.Dir == store.DirInternal {
		country = "__internal__"
	}
	proto := "tcp"
	if r.Proto == 17 {
		proto = "udp"
	} else if r.Proto != 6 {
		proto = strconv.Itoa(int(r.Proto))
	}
	dims := []store.DimKey{
		{TS: hour, Dim: "app", Val: r.App},
		{TS: hour, Dim: "port", Val: fmt.Sprintf("%d/%s\t%s", r.SvcPort, proto, r.App)},
		{TS: hour, Dim: "country", Val: country},
		{TS: hour, Dim: "proto", Val: strconv.Itoa(int(r.Proto))},
		{TS: hour, Dim: "dir", Val: strconv.Itoa(int(r.Dir))},
		{TS: hour, Dim: "exporter", Val: r.Exporter},
		{TS: hour, Dim: "segment", Val: r.Segment},
		{TS: hour, Dim: "encap", Val: strconv.Itoa(int(r.Encap))},
	}
	if r.PeerASN != 0 {
		dims = append(dims, store.DimKey{TS: hour, Dim: "asn", Val: fmt.Sprintf("%d\t%s", r.PeerASN, r.PeerOrg)})
	}
	for _, k := range dims {
		add(p.rDim, k)
	}
	_ = hasPeer
}

// The pipeline tracks how late data arrives: the time from the start of a
// flow to its arrival here, weighted by its bytes. Until a flow is exported
// the minutes it covers are incomplete. sFlow arrives at once; NetFlow and
// IPFIX after the device's active timeout. The delay used is the one by
// which 97% of the bytes have arrived, over the last two minutes.
var delayEdges = []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second, 20 * time.Second, 30 * time.Second,
	45 * time.Second, time.Minute, 90 * time.Second, 2 * time.Minute}

const maxSettle = 2 * time.Minute

type delayHist struct{ cur, prev [10]float64 }

func (p *Pipeline) noteDelay(recs []flow.Record, now time.Time) {
	p.dmu.Lock()
	defer p.dmu.Unlock()
	for i := range recs {
		r := &recs[i]
		d, b := now.Sub(r.Start), float64(r.Bytes)*max(r.Mult, 1)
		k := len(delayEdges)
		for j, e := range delayEdges {
			if d <= e {
				k = j
				break
			}
		}
		p.delays.cur[k] += max(b, 1)
	}
}

// updateSettle tells the store how far back data is complete.
func (p *Pipeline) updateSettle(time.Time) {
	p.dmu.Lock()
	h := p.delays
	p.delays.prev, p.delays.cur = p.delays.cur, [10]float64{}
	p.dmu.Unlock()
	var sum [10]float64
	tot := 0.0
	for k := range sum {
		sum[k] = h.cur[k] + h.prev[k]
		tot += sum[k]
	}
	if tot == 0 {
		return
	}
	s, acc := maxSettle, 0.0
	for k, e := range delayEdges {
		acc += sum[k]
		if acc >= 0.97*tot {
			s = e
			break
		}
	}
	store.SetSettle(s + time.Second)
}
