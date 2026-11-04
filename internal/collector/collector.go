// Package collector receives sFlow, NetFlow and IPFIX over UDP, decodes it
// and hands records to the pipeline. Any listener accepts any of the
// protocols; the version field decides.
package collector

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net"
	"net/netip"
	"runtime/debug"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/githubflyideas/traffic66/internal/decode/nf"
	"github.com/githubflyideas/traffic66/internal/decode/sflow"
	"github.com/githubflyideas/traffic66/internal/flow"
)

// Sink receives decoded data.
type Sink interface {
	Submit([]flow.Record)
	SubmitCounters([]flow.IfCounters)
}

// Listener is one UDP socket.
type Listener struct {
	Addr      string
	Proto     string // the protocol the port is meant for; informational
	conn      *net.UDPConn
	RcvBuf    int // bytes the OS granted
	Packets   atomic.Uint64
	Bytes     atomic.Uint64
	Undecoded atomic.Uint64
	Dropped   atomic.Uint64 // datagrams from a blocked source
	Panics    atomic.Uint64 // datagrams whose decode panicked
	LastErr   atomic.Value
}

// Collector owns listeners and decoder state.
type Collector struct {
	sink Sink
	NF   *nf.Decoder

	mu        sync.Mutex
	sf        map[sfKey]*sfState
	sfAgents  map[netip.Addr]*sfAgent
	Listeners []*Listener
	Skewed    atomic.Uint64
	// Block drops datagrams from the sources in it, before anything is
	// decoded. Nil means nothing is blocked.
	Block *Blocklist
	// Live marks records with implausible timestamps for correction.
	Live bool
}

type sfKey struct {
	agent    netip.Addr
	subAgent uint32
	source   uint32
}

type sfState struct {
	lastPool, lastSeq, lastDrops uint32
	init                         bool
	winPool, winRecv             float64
	evalAt                       time.Time
	eff                          float64
	rate                         uint32
	lost, drops                  uint64

	// ref and refSeq are the pool and sequence number of the newest
	// sample; checks and sum say whether the pool counts packets the way
	// it should, from the first samples
	ref    uint32
	refSeq uint32
	checks int
	sum    float64
	poolOK bool

	// streams: some agents run more than one sampler under one source id
	// (one per direction, say), each with its own pool and sequence
	streams []sfStream
	tick    uint64
	// dlost is how many of the agent's datagrams were lost so far, when
	// known: a run of samples can only have been lost with some of them
	dlost  uint64
	dknown bool
	dper   int // most samples the agent put in one datagram

	// recent are the last records of this source, which the samples lost
	// before a newer one are counted as
	recent []flow.Record
	next   int
}

// sfStream is one sampler's counters: the newest pool and sequence number,
// the sequence numbers it skipped (a reordered sample may still bring one)
// and the ones it recently had (one delivered twice adds nothing).
type sfStream struct {
	pool, seq, drops uint32
	dlost            uint64
	missing          []uint32
	seen             [streamSeen]struct{ seq, pool uint32 }
	nseen            int
	used             uint64
}

const (
	maxStreams = 4
	streamSeen = 64
)

func (h *sfStream) saw(seq, pool uint32) {
	h.seen[h.nseen%streamSeen] = struct{ seq, pool uint32 }{seq, pool}
	h.nseen++
}

func (h *sfStream) had(seq, pool uint32) bool {
	for i := 0; i < min(h.nseen, streamSeen); i++ {
		if h.seen[i].seq == seq && h.seen[i].pool == pool {
			return true
		}
	}
	return false
}

// recentKept is how many records of a source the lost samples are spread
// over.
const recentKept = 32

func (st *sfState) remember(r flow.Record) {
	if len(st.recent) < recentKept {
		st.recent = append(st.recent, r)
		return
	}
	st.recent[st.next] = r
	st.next = (st.next + 1) % recentKept
}

type sfAgent struct {
	addr      netip.Addr
	datagrams uint64
	samples   uint64
	counters  uint64
	lostDgram uint64
	lastSeq   map[uint32]uint32
	lastSeen  time.Time
	firstSeen time.Time // the agent's first datagram
	decodeErr uint64
	perDgram  int // most flow samples seen in one datagram
}

func New(sink Sink) *Collector {
	return &Collector{sink: sink, NF: nf.NewDecoder(), sf: map[sfKey]*sfState{}, sfAgents: map[netip.Addr]*sfAgent{}, Live: true}
}

// Listen opens a UDP socket; proto is only a label.
func (c *Collector) Listen(addr, proto string) (*Listener, error) {
	ua, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", ua)
	if err != nil {
		return nil, fmt.Errorf("listen %s (%s): %w", addr, proto, err)
	}
	want := 16 << 20
	conn.SetReadBuffer(want)
	if readBufSize(conn) < want {
		forceReadBuf(conn, want)
	}
	l := &Listener{Addr: addr, Proto: proto, conn: conn}
	l.RcvBuf = readBufSize(conn)
	if l.RcvBuf > 0 && l.RcvBuf < 4<<20 {
		log.Printf("collector: %s receive buffer is only %d KB; bursts may be lost (raise net.core.rmem_max on Linux)", addr, l.RcvBuf/1024)
	}
	c.mu.Lock()
	c.Listeners = append(c.Listeners, l)
	c.mu.Unlock()
	return l, nil
}

// LocalAddr is the bound address (useful when the port was 0).
func (l *Listener) LocalAddr() string { return l.conn.LocalAddr().String() }

// Serve reads from l until ctx ends.
func (c *Collector) Serve(ctx context.Context, l *Listener) {
	go func() { <-ctx.Done(); l.conn.Close() }()
	buf := make([]byte, 65535)
	for {
		n, from, err := l.conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			l.LastErr.Store(err.Error())
			time.Sleep(10 * time.Millisecond)
			continue
		}
		l.Packets.Add(1)
		l.Bytes.Add(uint64(n))
		// Blocked senders are dropped here, before the decoder: nothing
		// is parsed, nothing is stored, nothing reaches the rules. The
		// packet is still counted above, because it did arrive on the
		// socket and a listener that under-reports what reached it is
		// harder to diagnose than one that reports a drop.
		if c.Block != nil && c.Block.Has(from.Addr().Unmap()) {
			l.Dropped.Add(1)
			continue
		}
		if !c.handleSafely(l, buf[:n], from.Addr().Unmap(), time.Now()) {
			l.Undecoded.Add(1)
		}
	}
}

// handleSafely keeps one datagram from taking the process down with it.
// A panic in a decoder is a bug and the log says so, with a stack; but
// anyone who can reach the port can send the packet that triggers it, and
// an analyser that dies on a packet it cannot parse is worse than one that
// counts it and reads the next. The datagram is counted as undecoded too,
// because it was.
func (c *Collector) handleSafely(l *Listener, b []byte, from netip.Addr, now time.Time) (ok bool) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		ok = false
		l.Panics.Add(1)
		l.LastErr.Store(fmt.Sprintf("panic decoding %d bytes from %s: %v", len(b), from, r))
		log.Printf("collector: panic decoding %d bytes from %s: %v\n%s", len(b), from, r, debug.Stack())
	}()
	return c.Handle(b, from, now)
}

// Handle decodes one datagram. It reports whether it was understood.
func (c *Collector) Handle(b []byte, from netip.Addr, now time.Time) bool {
	if len(b) < 4 {
		return false
	}
	if binary.BigEndian.Uint32(b) == 5 {
		return c.handleSFlow(b, from, now)
	}
	res, err := c.NF.Decode(b, from, now)
	if err != nil {
		return false
	}
	c.fixTimes(res.Records, now)
	c.sink.Submit(res.Records)
	return true
}

// Tick releases records held for a sampling rate; call periodically.
// maxAgents caps tracked sFlow agents; agentIdle is how long a silent
// agent is remembered.
const (
	maxAgents = 10000
	agentIdle = 24 * time.Hour
)

func (c *Collector) Tick(now time.Time) {
	c.mu.Lock()
	for a, ag := range c.sfAgents {
		if now.Sub(ag.lastSeen) > agentIdle {
			delete(c.sfAgents, a)
		}
	}
	for k := range c.sf {
		if c.sfAgents[k.agent] == nil {
			delete(c.sf, k)
		}
	}
	c.mu.Unlock()
	// The drop counts of blocked sources live in memory on the receive
	// path; this is where they reach the file.
	if c.Block != nil {
		if err := c.Block.Flush(); err != nil {
			log.Printf("collector: blocklist: %v", err)
		}
	}
	res := c.NF.Flush(now)
	c.fixTimes(res.Records, now)
	c.sink.Submit(res.Records)
}

// fixTimes moves records whose exporter clock is far off to receive time,
// preserving their duration.
func (c *Collector) fixTimes(recs []flow.Record, now time.Time) {
	if !c.Live {
		return
	}
	for i := range recs {
		r := &recs[i]
		if r.End.After(now.Add(2*time.Minute)) || r.End.Before(now.Add(-3*time.Hour)) {
			d := now.Sub(r.End)
			r.End = r.End.Add(d)
			r.Start = r.Start.Add(d)
			c.Skewed.Add(1)
		}
	}
}

func (c *Collector) handleSFlow(b []byte, from netip.Addr, now time.Time) bool {
	d, err := sflow.Decode(b, now)
	c.mu.Lock()
	defer c.mu.Unlock()
	agentAddr := from
	if d != nil && d.Agent.IsValid() {
		agentAddr = d.Agent.Unmap()
	}
	ag := c.sfAgents[agentAddr]
	if ag == nil {
		if len(c.sfAgents) >= maxAgents {
			return false // too many agents (spoofed sources?); counted as undecoded
		}
		ag = &sfAgent{addr: agentAddr, lastSeq: map[uint32]uint32{}}
		c.sfAgents[agentAddr] = ag
	}
	ag.lastSeen = now
	if ag.firstSeen.IsZero() {
		ag.firstSeen = now
	}
	if d == nil {
		ag.decodeErr++
		return false
	}
	if err != nil {
		ag.decodeErr++
	}
	ag.datagrams++
	ag.perDgram = max(ag.perDgram, len(d.Flows))
	if last, ok := ag.lastSeq[d.SubAgent]; ok {
		if gap := d.Seq - last - 1; gap > 0 && gap < 1<<20 {
			ag.lostDgram += uint64(gap)
		}
	}
	ag.lastSeq[d.SubAgent] = d.Seq

	recs := make([]flow.Record, 0, len(d.Flows))
	for i := range d.Flows {
		s := &d.Flows[i]
		ag.samples++
		st := c.sf[sfKey{agentAddr, d.SubAgent, s.SourceID}]
		if st == nil {
			st = &sfState{evalAt: now}
			c.sf[sfKey{agentAddr, d.SubAgent, s.SourceID}] = st
		}
		st.dlost, st.dknown, st.dper = ag.lostDgram, true, ag.perDgram
		w, gap := st.update(s, now)
		// The samples lost before this one, and one whose packet could
		// not be read, count as the source's recent records rather than
		// all as this one: losses come in runs, in bursts, and the next
		// sample to arrive is often from after the burst.
		own, spread := w, 0.0
		if gap > 0 && len(st.recent) > 0 {
			own = w / float64(gap+1)
			spread = w - own
		}
		if !s.Decoded {
			spread, own = spread+own, 0
		}
		if spread > 0 && len(st.recent) > 0 {
			m := spread / float64(len(st.recent))
			for _, r := range st.recent {
				r.Mult = m
				recs = append(recs, r)
			}
		}
		if own == 0 {
			continue
		}
		r := s.Record
		r.Exporter = agentAddr
		r.Mult = own
		recs = append(recs, r)
		st.remember(r)
	}
	for i := range d.Counters {
		d.Counters[i].Exporter = agentAddr
	}
	ag.counters += uint64(len(d.Counters))
	c.sink.Submit(recs)
	c.sink.SubmitCounters(d.Counters)
	return true
}

// update tracks the effective sampling rate: packets offered for sampling
// (sample_pool) per sample actually received, shown on the sources page.
// It returns what the sample counts for: the packets the pool advanced by
// since the newest sample before it, which covers the samples lost or
// dropped on the way; 0 for a sample older than one already counted; the
// configured rate where the pool cannot be used; and how many samples were
// lost just before it.
func (st *sfState) update(s *sflow.FlowSample, now time.Time) (float64, uint32) {
	w, gap, h, ahead := st.weight(s)
	if h == nil {
		// late or repeated: already accounted for
		return w, gap
	}
	if st.init && s.Rate != st.rate {
		// The agent's configured rate changed. Everything measured so far
		// was measured against the old one, and the effective rate is only
		// recomputed every 30 seconds and after 30 more samples, so
		// carrying it over scaled the new samples by the old rate for up to
		// half a minute — ten times the traffic when 1 in 1000 became 1 in
		// 100, and a tenth of it the other way. Dropping it falls straight
		// back to the rate the agent now declares, which the decoder has
		// already put on the record, until there is enough to measure
		// again.
		st.eff = 0
		st.winPool, st.winRecv = 0, 0
		st.evalAt = now
	}
	st.rate = s.Rate
	if ahead > 0 {
		st.winPool += float64(ahead)
		st.winRecv++
		if dd := s.Drops - h.drops; dd < 1<<31 {
			st.drops += uint64(dd)
		}
	}
	h.drops = s.Drops
	st.init = true
	st.lastPool, st.lastSeq, st.lastDrops = s.Pool, s.Seq, s.Drops
	if now.Sub(st.evalAt) >= 30*time.Second && st.winRecv >= 30 {
		e := st.winPool / st.winRecv
		cfg := float64(s.Rate)
		if cfg > 0 && (e < cfg*0.5 || e > cfg*4) {
			e = cfg // implausible: agent resets or pool semantics differ
		}
		st.eff = e
		st.winPool, st.winRecv = 0, 0
		st.evalAt = now
	}
	return w, gap
}

// weight is what a sample counts for and how many samples of its sampler
// were lost just before it. h is the sampler it moved forward (nil for a
// late or repeated one) and ahead how far its pool moved.
func (st *sfState) weight(s *sflow.FlowSample) (w float64, gap uint32, h *sfStream, ahead uint32) {
	rate := float64(max(s.Rate, 1))
	limit := rate * 1000
	st.tick++
	for i := range st.streams {
		c := &st.streams[i]
		// the same sample again
		if c.had(s.Seq, s.Pool) {
			return 0, 0, nil, 0
		}
		// late: a sequence number this sampler skipped, its pool a little
		// behind; the newer sample already counted its packets
		if k := slices.Index(c.missing, s.Seq); k >= 0 && c.pool-s.Pool < 1<<31 && float64(c.pool-s.Pool) <= limit {
			c.missing = slices.Delete(c.missing, k, k+1)
			c.saw(s.Seq, s.Pool)
			if !st.poolOK {
				return rate, 0, nil, 0
			}
			return 0, 0, nil, 0
		}
	}
	// the sampler this sample moves forward: its sequence number and pool
	// both ahead, the pool by about the rate per sequence number (a run of
	// lost samples moves the sequence number as far); of several, the one
	// whose pool moved closest to that. A jump of more than 1000 is taken
	// for another sampler rather than a loss: the two look the same.
	best, bestOff := -1, 0.0
	for i := range st.streams {
		c := &st.streams[i]
		sa, pa := s.Seq-c.seq, s.Pool-c.pool
		// samples skipped can only have been lost in the datagrams lost
		// or the drops the agent counted; more, and the sample is another
		// sampler's
		could := uint64(s.Drops - c.drops)
		if could >= 1<<31 {
			could = 0
		}
		could += (st.dlost - c.dlost) * uint64(max(st.dper, 1))
		if sa >= 1 && sa < 1<<20 && (!st.dknown || uint64(sa-1) <= could) && pa >= 1 && pa < 1<<31 && (float64(pa) <= limit || float64(pa) <= rate*float64(sa)*20) {
			if off := math.Abs(float64(pa) - rate*float64(sa)); best < 0 || off < bestOff {
				best, bestOff = i, off
			}
		}
	}
	if best >= 0 {
		h = &st.streams[best]
		sa, pa := s.Seq-h.seq, s.Pool-h.pool
		if sa > 1 && sa <= streamSeen {
			for q := h.seq + 1; q != s.Seq; q++ {
				h.missing = append(h.missing, q)
			}
			if n := len(h.missing); n > streamSeen {
				h.missing = append(h.missing[:0], h.missing[n-streamSeen:]...)
			}
		}
		h.pool, h.seq, h.used, h.dlost = s.Pool, s.Seq, st.tick, st.dlost
		h.saw(s.Seq, s.Pool)
		st.ref, st.refSeq = s.Pool, s.Seq
		gap = sa - 1
		st.lost += uint64(gap)
		if !st.poolOK {
			// the first ten samples show whether the pool counts packets
			// (some agents leave it at 0 or count samples instead)
			if st.checks < 10 {
				st.checks++
				st.sum += float64(pa) / float64(sa)
				if st.checks == 10 {
					m := st.sum / 10
					st.poolOK = m >= rate*0.5 && m <= rate*20
				}
			}
			return rate, 0, h, pa
		}
		return float64(pa), gap, h, pa
	}
	// a sampler not seen yet: another one under this source id, or the
	// agent restarted; it counts at the configured rate until it moves
	n := sfStream{pool: s.Pool, seq: s.Seq, drops: s.Drops, used: st.tick, dlost: st.dlost}
	n.saw(s.Seq, s.Pool)
	if len(st.streams) < maxStreams {
		st.streams = append(st.streams, n)
		h = &st.streams[len(st.streams)-1]
	} else {
		old := 0
		for i := range st.streams {
			if st.streams[i].used < st.streams[old].used {
				old = i
			}
		}
		st.streams[old] = n
		h = &st.streams[old]
	}
	st.ref, st.refSeq = s.Pool, s.Seq
	return rate, 0, h, 0
}

// SourceInfo summarises one exporter for the UI.
type SourceInfo struct {
	Exporter    string  `json:"exporter"`
	Proto       string  `json:"proto"`
	Domain      uint32  `json:"domain"`
	Packets     uint64  `json:"packets"`
	Records     uint64  `json:"records"`
	LostPct     float64 `json:"lost_pct"`
	LostRecords uint64  `json:"lost_records"`
	// the loss split by where it happened: on the way (gaps in sequence
	// numbers) and in the device itself (sFlow's own drop counter)
	TransitPct    float64   `json:"transit_pct"`
	DropPct       float64   `json:"drop_pct"`
	Sampling      string    `json:"sampling"`
	SamplingState string    `json:"sampling_state"`
	Effective     float64   `json:"effective,omitempty"`
	LossComp      float64   `json:"loss_comp,omitempty"`
	Pending       int       `json:"pending"`
	NoTemplate    uint64    `json:"no_template"`
	Templates     int       `json:"templates"`
	Counters      uint64    `json:"counters"`
	DecodeErrors  uint64    `json:"decode_errors"`
	LastSeen      time.Time `json:"last_seen"`
	FirstSeen     time.Time `json:"first_seen,omitempty"`
	// WaitStep is what this exporter's data is held for: "templates" or
	// "sampling"; empty when nothing is held.
	WaitStep  string        `json:"wait_step,omitempty"`
	ClockSkew time.Duration `json:"clock_skew_ns"`
	// NetFlow/IPFIX: the templates received (id and field ids), shown when
	// the sampling rate is not declared
	TemplateInfo []string `json:"template_info,omitempty"`
	// NetFlow/IPFIX: whether the templates carry flowDirection ("yes"/"no")
	DirField string `json:"dir_field,omitempty"`
}

// WaitStep names the stage an exporter's records are held at: templates
// first (nothing can be read without one), then the sampling rate.
func WaitStep(templates int, noTemplate uint64, samplingState string) string {
	switch {
	case templates == 0 && noTemplate > 0:
		return "templates"
	case samplingState == "waiting":
		return "sampling"
	}
	return ""
}

// Sources lists every exporter seen.
func (c *Collector) Sources() []SourceInfo {
	var out []SourceInfo
	for _, s := range c.NF.Stats() {
		lost := 0.0
		if t := s.Records + s.LostRecords; t > 0 {
			lost = float64(s.LostRecords) / float64(t) * 100
		}
		samp := ""
		for i, r := range s.Rates {
			if i == 3 {
				samp += "…"
				break
			}
			if i > 0 {
				samp += ", "
			}
			samp += fmt.Sprintf("1:%d", r)
		}
		if samp == "" && s.SamplingState == "assumed1" {
			samp = "1:1"
		}
		out = append(out, SourceInfo{
			Exporter: s.Exporter.Unmap().String(), Proto: s.Source.String(), Domain: s.Domain,
			Packets: s.Packets, Records: s.Records, LostPct: lost, LostRecords: s.LostRecords, TransitPct: lost,
			Sampling: samp, SamplingState: s.SamplingState, Pending: s.Pending, NoTemplate: s.NoTemplate,
			Templates: s.Templates, LastSeen: s.LastSeen, ClockSkew: s.ClockSkew, LossComp: s.LossComp,
			TemplateInfo: s.TemplateInfo, DirField: s.DirField, FirstSeen: s.FirstSeen,
			WaitStep: WaitStep(s.Templates, s.NoTemplate, s.SamplingState),
		})
	}
	c.mu.Lock()
	for _, ag := range c.sfAgents {
		si := SourceInfo{Exporter: ag.addr.String(), Proto: flow.SrcSFlow.String(), Packets: ag.datagrams, Records: ag.samples,
			Counters: ag.counters, DecodeErrors: ag.decodeErr, LastSeen: ag.lastSeen, FirstSeen: ag.firstSeen, SamplingState: "known"}
		var lost, drops uint64
		rates := map[uint32]bool{}
		var effSum, effN float64
		for k, st := range c.sf {
			if k.agent != ag.addr {
				continue
			}
			lost += st.lost
			drops += st.drops
			rates[st.rate] = true
			if st.eff > 0 {
				effSum += st.eff
				effN++
			}
		}
		si.LostRecords = lost + drops
		if t := ag.samples + si.LostRecords; t > 0 {
			si.LostPct = float64(si.LostRecords) / float64(t) * 100
			si.TransitPct = float64(lost) / float64(t) * 100
			si.DropPct = float64(drops) / float64(t) * 100
		}
		var rs []int
		for r := range rates {
			rs = append(rs, int(r))
		}
		sort.Ints(rs)
		for i, r := range rs {
			if i > 0 {
				si.Sampling += ", "
			}
			si.Sampling += fmt.Sprintf("1:%d", r)
			if i == 3 {
				si.Sampling += "…"
				break
			}
		}
		if effN > 0 {
			si.Effective = effSum / effN
		}
		out = append(out, si)
	}
	c.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Exporter < out[j].Exporter })
	return out
}

// SetSampling applies the names' sampling settings: devices that export
// every packet, and hand-set rates for devices that do not declare one.
// Records that were waiting for a rate go on with it.
func (c *Collector) SetSampling(unsampled map[netip.Addr]bool, manual map[netip.Addr]uint32) {
	c.NF.SetUnsampled(unsampled)
	if res := c.NF.SetManual(manual); len(res.Records) > 0 {
		c.sink.Submit(res.Records)
	}
}
