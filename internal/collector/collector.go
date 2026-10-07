// Package collector receives sFlow, NetFlow and IPFIX over UDP, decodes it
// and hands records to the pipeline. Any listener accepts any of the
// protocols; the version field decides.
package collector

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/netip"
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
}

type sfAgent struct {
	addr      netip.Addr
	datagrams uint64
	samples   uint64
	counters  uint64
	lostDgram uint64
	lastSeq   map[uint32]uint32
	lastSeen  time.Time
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
		if !c.Handle(buf[:n], from.Addr().Unmap(), time.Now()) {
			l.Undecoded.Add(1)
		}
	}
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
	if d == nil {
		ag.decodeErr++
		return false
	}
	if err != nil {
		ag.decodeErr++
	}
	ag.datagrams++
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
		st.update(s, now)
		if !s.Decoded {
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

// update tracks the effective sampling rate: packets offered for sampling
// (sample_pool) per sample actually received. Lost datagrams and samples
// dropped by the agent both raise it, compensating for the missing data.
func (st *sfState) update(s *sflow.FlowSample, now time.Time) {
	st.rate = s.Rate
	if st.init {
		dp := s.Pool - st.lastPool
		if dp < 1<<31 {
			st.winPool += float64(dp)
			st.winRecv++
		}
		if gap := s.Seq - st.lastSeq - 1; gap > 0 && gap < 1<<20 {
			st.lost += uint64(gap)
		}
		if dd := s.Drops - st.lastDrops; dd < 1<<31 {
			st.drops += uint64(dd)
		}
	}
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
	TransitPct    float64       `json:"transit_pct"`
	DropPct       float64       `json:"drop_pct"`
	Sampling      string        `json:"sampling"`
	SamplingState string        `json:"sampling_state"`
	Effective     float64       `json:"effective,omitempty"`
	LossComp      float64       `json:"loss_comp,omitempty"`
	Pending       int           `json:"pending"`
	NoTemplate    uint64        `json:"no_template"`
	Templates     int           `json:"templates"`
	Counters      uint64        `json:"counters"`
	DecodeErrors  uint64        `json:"decode_errors"`
	LastSeen      time.Time     `json:"last_seen"`
	ClockSkew     time.Duration `json:"clock_skew_ns"`
	// NetFlow/IPFIX: the templates received (id and field ids), shown when
	// the sampling rate is not declared
	TemplateInfo []string `json:"template_info,omitempty"`
	// NetFlow/IPFIX: whether the templates carry flowDirection ("yes"/"no")
	DirField string `json:"dir_field,omitempty"`
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
			TemplateInfo: s.TemplateInfo, DirField: s.DirField,
		})
	}
	c.mu.Lock()
	for _, ag := range c.sfAgents {
		si := SourceInfo{Exporter: ag.addr.String(), Proto: flow.SrcSFlow.String(), Packets: ag.datagrams, Records: ag.samples,
			Counters: ag.counters, DecodeErrors: ag.decodeErr, LastSeen: ag.lastSeen, SamplingState: "known"}
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
