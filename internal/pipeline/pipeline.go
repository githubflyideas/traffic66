// Package pipeline turns flow records into stored rows: it enriches them,
// normalises byte counts to the link layer, spreads long flows over the
// minutes they covered, marks duplicate reports and maintains rollups.
package pipeline

import (
	"context"
	"fmt"
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
	cfg Config
	st  *store.Store
	Inv *enrich.Inventory
	ASN *enrich.ASNDB
	Thr *enrich.Threats

	in  chan []flow.Record
	ctr chan []flow.IfCounters

	mu    sync.Mutex
	batch []store.Row
	rTS   map[store.TSKey]*store.Counters
	rHost map[store.HostKey]*store.Counters
	rDim  map[store.DimKey]*store.Counters
	dedup map[time.Time]map[dkey]origin
	order []time.Time

	Records   atomic.Uint64
	Rows      atomic.Uint64
	DupRows   atomic.Uint64
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
		dedup: map[time.Time]map[dkey]origin{},
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
		case <-rollTick.C:
			p.FlushRollups()
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

	slices := minuteSlices(r.Start, r.End)
	var bLeft, wLeft, pLeft = l3, wire, pkts
	for i, sl := range slices {
		row := base
		row.TS = sl.t
		last := i == len(slices)-1
		if last {
			row.Bytes, row.Wire, row.Pkts = uint64(math.Round(bLeft)), uint64(math.Round(wLeft)), uint64(math.Round(pLeft))
			row.Flows = 1
		} else {
			b, w, pk := l3*sl.frac, wire*sl.frac, pkts*sl.frac
			row.Bytes, row.Wire, row.Pkts = uint64(math.Round(b)), uint64(math.Round(w)), uint64(math.Round(pk))
			bLeft -= float64(row.Bytes)
			wLeft -= float64(row.Wire)
			pLeft -= float64(row.Pkts)
		}
		row.Dup = p.isDup(sl.t, key, org2)
		p.batch = append(p.batch, row)
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

// isDup reports whether another observation point already reported this
// packet direction in this minute.
func (p *Pipeline) isDup(t time.Time, k dkey, o origin) bool {
	m := p.dedup[t]
	if m == nil {
		m = map[dkey]origin{}
		p.dedup[t] = m
		p.order = append(p.order, t)
		for len(p.order) > 8 {
			delete(p.dedup, p.order[0])
			p.order = p.order[1:]
		}
	}
	prev, ok := m[k]
	if !ok {
		m[k] = o
		return false
	}
	return prev != o
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
