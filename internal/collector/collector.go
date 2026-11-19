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
	// of the newest sample: the highest sequence number seen
	lastPool, lastSeq, lastDrops uint32
	init                         bool
	// the last sfWin sequence numbers and their pools, to tell a sample
	// that arrives late or twice from one that was lost
	seen     [sfWin]sfSeen
	seenMask uint64
	// since evalAt: samples received and missing, and the pool's growth
	// over winPoolN received samples
	winMiss, winRecv  float64
	winPool, winPoolN float64
	winGap            uint64 // samples missing from sequence gaps
	shared            bool   // the gaps are not losses: see update
	evalAt            time.Time
	eff               float64
	rate              uint32
	lost, drops       uint64
}

type sfSeen struct{ seq, pool uint32 }

const (
	// sfWin is how far back a sample may arrive late (or again) and still
	// be recognised; UDP reorders over a few datagrams, not dozens.
	sfWin = 64
	// sfOutage is the largest gap that counts towards the loss rate: a
	// longer one is an outage, whose traffic is gone, not a loss rate that
	// the next samples should make up for.
	sfOutage = 256
	// sfMaxComp caps the compensation: 95% of the samples lost.
	sfMaxComp = 20
)

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
			// Move the record to receive time and keep its duration. Times
			// centuries off (microseconds in a millisecond field) cannot be
			// shifted by a time.Duration, which saturates at 292 years, so
			// the end is set rather than moved.
			d := r.End.Sub(r.Start)
			if d < 0 || d > maxFlowSpan {
				d = 0
			}
			r.End = now
			r.Start = now.Add(-d)
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
	if last, ok := ag.lastSeq[d.SubAgent]; !ok {
		ag.lastSeq[d.SubAgent] = d.Seq
	} else if n := int32(d.Seq - last); n > 0 && n < 1<<20 {
		ag.lostDgram += uint64(n - 1)
		ag.lastSeq[d.SubAgent] = d.Seq
	} else if n < 0 && n > -sfWin {
		// a datagram overtaken by a later one, already counted as lost
		if ag.lostDgram > 0 {
			ag.lostDgram--
		}
	} else if n != 0 {
		ag.lastSeq[d.SubAgent] = d.Seq // the agent started over
	}

	recs := make([]flow.Record, 0, len(d.Flows))
	for i := range d.Flows {
		s := &d.Flows[i]
		ag.samples++
		st := c.sf[sfKey{agentAddr, d.SubAgent, s.SourceID}]
		if st == nil {
			st = &sfState{evalAt: now}
			c.sf[sfKey{agentAddr, d.SubAgent, s.SourceID}] = st
		}
		if !st.update(s, now) || !s.Decoded {
			continue
		}
		r := s.Record
		r.Exporter = agentAddr
		if st.eff > 0 {
			r.Mult = st.eff
		}
		recs = append(recs, r)
	}
	for i := range d.Counters {
		d.Counters[i].Exporter = agentAddr
	}
	ag.counters += uint64(len(d.Counters))
	c.sink.Submit(recs)
	c.sink.SubmitCounters(d.Counters)
	return true
}

// update tracks the effective sampling rate: the configured rate, raised
// by the samples that never arrived, lost on the way (gaps in the sequence
// numbers) or dropped by the agent itself (its drop counter). It reports
// whether the sample is to be counted: one delivered twice is not.
//
// The agent's sample_pool is not the measure: vendors count it differently
// (H3C among them), and a rate derived from it scaled a 1:10240 port by
// about 1:12200, a fifth more traffic than the interface carried. It is the
// check on the sequence numbers instead: some agents number the samples of
// all their interfaces in one sequence, so that each interface sees gaps
// that are not losses; where the pool grew no faster than the declared rate
// while the gaps say most samples went missing, the pool is believed.
func (st *sfState) update(s *sflow.FlowSample, now time.Time) bool {
	if !st.init {
		st.init, st.rate = true, s.Rate
		st.restart(s)
		st.winRecv++
		return true
	}
	if s.Rate != st.rate {
		// The configured rate changed. The share of samples lost does not
		// depend on it, so the measured compensation carries over to the
		// new rate; only the pool, which grew at the old rate, starts over.
		if st.eff > 0 && st.rate > 0 {
			st.eff = st.eff / float64(st.rate) * float64(s.Rate)
		}
		st.rate = s.Rate
		st.winPool, st.winPoolN = 0, 0
	}
	d := int32(s.Seq - st.lastSeq)
	switch {
	case d <= 0 && d > -sfWin && st.saw(s.Seq):
		if st.seen[s.Seq%sfWin].pool == s.Pool {
			return false // the same sample delivered again
		}
		st.restart(s) // the same number for another sample: the agent started over
	case d < 0 && d > -sfWin:
		// arrived after a later one, which counted it as lost
		st.mark(s)
		if st.winGap > 0 {
			st.winGap--
			st.winMiss--
			if !st.shared && st.lost > 0 {
				st.lost--
			}
		}
		st.winPoolN++
	case d <= 0 || d >= 1<<20:
		st.restart(s) // the sequence started over: a restarted agent
	default:
		gap := uint32(d) - 1
		if !st.shared {
			st.lost += uint64(gap)
		}
		if dd := s.Drops - st.lastDrops; dd < 1<<31 {
			st.drops += uint64(dd)
			st.winMiss += float64(dd)
		}
		if gap <= sfOutage {
			st.winGap += uint64(gap)
			st.winMiss += float64(gap)
			if dp := s.Pool - st.lastPool; dp < 1<<31 {
				st.winPool += float64(dp)
				st.winPoolN++
			}
		}
		st.lastPool, st.lastSeq, st.lastDrops = s.Pool, s.Seq, s.Drops
		st.mark(s)
	}
	st.winRecv++
	if now.Sub(st.evalAt) >= 30*time.Second && st.winRecv >= 30 {
		cfg := float64(st.rate)
		e := cfg * (st.winRecv + st.winMiss) / st.winRecv
		shared := false
		if st.winPoolN >= 30 && cfg > 0 {
			p := st.winPool / st.winPoolN
			// the margin leaves room for the pool's own spread over few samples
			if p >= cfg/2 && e > p*(1.5+2/math.Sqrt(st.winPoolN)) {
				e, shared = math.Max(cfg, p), true
			}
		}
		// the gaps of this window were counted as lost, or not, on the
		// last verdict; this one decides
		if shared && !st.shared {
			st.lost -= min(st.lost, st.winGap)
		} else if !shared && st.shared {
			st.lost += st.winGap
		}
		st.shared = shared
		st.eff = min(e, cfg*sfMaxComp)
		st.winMiss, st.winRecv, st.winPool, st.winPoolN, st.winGap = 0, 0, 0, 0, 0
		st.evalAt = now
	}
	return true
}

// restart takes s as the first sample of a new sequence.
func (st *sfState) restart(s *sflow.FlowSample) {
	st.lastPool, st.lastSeq, st.lastDrops = s.Pool, s.Seq, s.Drops
	st.seenMask = 0
	st.mark(s)
}

func (st *sfState) mark(s *sflow.FlowSample) {
	i := s.Seq % sfWin
	st.seen[i] = sfSeen{s.Seq, s.Pool}
	st.seenMask |= 1 << i
}

func (st *sfState) saw(seq uint32) bool {
	i := seq % sfWin
	return st.seenMask&(1<<i) != 0 && st.seen[i].seq == seq
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

// maxFlowSpan is the longest duration a record keeps when its times are
// moved to receive time.
const maxFlowSpan = 24 * time.Hour
