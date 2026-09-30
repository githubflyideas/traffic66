package sim

import (
	"context"
	"math"
	"math/rand/v2"
	"net/netip"
	"sort"
	"sync"
	"time"

	"github.com/githubflyideas/traffic66/internal/encode"
	"github.com/githubflyideas/traffic66/internal/flow"
)

const (
	coreRate   = 1000
	switchRate = 4096
	branchRate = 100
)

// session is one client/server exchange during a tick.
type session struct {
	c                  *conv
	client, server     netip.Addr
	cport              uint16
	upBytes, downBytes float64
	start, end         time.Time
}

// Sim produces traffic from the model.
type Sim struct {
	m   *model
	rng *rand.Rand
	// SessionScale divides session counts (fewer, larger flows).
	SessionScale float64

	// cumulative interface counters on the switch: ifindex -> octets/pkts
	ctrIn, ctrOut map[uint32]float64
	pktIn, pktOut map[uint32]float64
	pool          map[uint32]uint32
	sampleSeq     map[uint32]uint32

	// true link-layer counters of the NetFlow/IPFIX devices, served over
	// SNMP by the demo agents
	devMu sync.Mutex
	dev   map[devIf]*ifCtr
}

type devIf struct {
	dev netip.Addr
	idx uint32
}

type ifCtr struct{ inOct, outOct, inPkt, outPkt float64 }

// link adds pk packets of frame bytes each, entering in and leaving out.
func (s *Sim) link(dev netip.Addr, in, out uint32, pk, frame float64) {
	s.devMu.Lock()
	defer s.devMu.Unlock()
	if s.dev == nil {
		s.dev = map[devIf]*ifCtr{}
	}
	get := func(i uint32) *ifCtr {
		c := s.dev[devIf{dev, i}]
		if c == nil {
			c = &ifCtr{}
			s.dev[devIf{dev, i}] = c
		}
		return c
	}
	a, b := get(in), get(out)
	a.inOct += pk * frame
	a.inPkt += pk
	b.outOct += pk * frame
	b.outPkt += pk
}

// DeviceCounters returns the current counters of one NetFlow/IPFIX device.
func (s *Sim) DeviceCounters(at time.Time, dev netip.Addr) []flow.IfCounters {
	s.devMu.Lock()
	defer s.devMu.Unlock()
	var out []flow.IfCounters
	for k, c := range s.dev {
		if k.dev != dev {
			continue
		}
		out = append(out, flow.IfCounters{Time: at, Exporter: dev, IfIndex: k.idx, Speed: ifSpeed(dev, k.idx),
			InOctets: uint64(c.inOct), OutOctets: uint64(c.outOct), InPkts: uint64(c.inPkt), OutPkts: uint64(c.outPkt)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].IfIndex < out[j].IfIndex })
	return out
}

func ifSpeed(dev netip.Addr, idx uint32) uint64 {
	switch {
	case dev == CoreRouter && idx == ifCoreLAN:
		return 10e9
	case dev == Branch:
		return 100e6
	case dev == CoreRouter && idx == ifCoreISPB:
		return 500e6
	}
	return 1e9
}

// New builds a simulator whose incident window is anchored to now.
func New(now time.Time, seed uint64) *Sim {
	return &Sim{m: newModel(now), rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)), SessionScale: 4,
		ctrIn: map[uint32]float64{}, ctrOut: map[uint32]float64{}, pktIn: map[uint32]float64{}, pktOut: map[uint32]float64{},
		pool: map[uint32]uint32{}, sampleSeq: map[uint32]uint32{}}
}

func (s *Sim) poisson(mean float64) int {
	if mean <= 0 {
		return 0
	}
	if mean > 50 {
		v := int(math.Round(mean + math.Sqrt(mean)*s.rng.NormFloat64()))
		if v < 0 {
			return 0
		}
		return v
	}
	l, k, p := math.Exp(-mean), 0, 1.0
	for {
		p *= s.rng.Float64()
		if p <= l {
			return k
		}
		k++
	}
}

// sessions generates the exchanges for [t, t+dt).
func (s *Sim) sessions(t time.Time, dt time.Duration, today bool) []session {
	h := hourOf(t)
	secs := dt.Seconds()
	var out []session
	add := func(c *conv, mbps float64) {
		if mbps <= 0 {
			return
		}
		total := mbps * 1e6 / 8 * secs * (0.9 + 0.2*s.rng.Float64())
		n := int(math.Ceil(float64(c.sessions) / s.SessionScale * math.Min(1, mbps/150)))
		if n < 1 {
			n = 1
		}
		weights := make([]float64, n)
		sum := 0.0
		for i := range weights {
			weights[i] = s.rng.ExpFloat64()
			sum += weights[i]
		}
		for i := 0; i < n; i++ {
			b := total * weights[i] / sum
			out = append(out, session{c: c, client: c.clients(s.rng), server: c.server, cport: uint16(32768 + s.rng.IntN(28000)),
				upBytes: b * c.upShare, downBytes: b * (1 - c.upShare), start: t, end: t.Add(dt)})
		}
	}
	for i := range s.m.convs {
		c := &s.m.convs[i]
		add(c, c.mbps(h, today))
	}
	if today {
		inc := s.m.incidentMbps(t)
		if inc > 0 {
			c := &conv{name: "incident", clients: one(financePC), server: extC2.addr, proto: 6, port: 443, upShare: 0.97, ipSize: 1420,
				via: viaCore | viaSwitchUp, sessions: 3, coreIf: ifCoreISPA}
			add(c, inc)
		}
	}
	return out
}

func (s *Sim) swPorts(se *session) (clientPort, serverPort uint32) {
	clientPort = ifSwAccess
	if se.client == backupHost {
		clientPort = ifSwServers
	}
	serverPort = ifSwUplink
	if se.c.via&viaSwitchSrv != 0 && se.server.Is4() && se.server.As4()[0] == 10 {
		serverPort = ifSwServers
	}
	return
}

// Records converts sessions to decoded records as exporters would report
// them (sampling applied), plus switch counter snapshots. Used for backfill.
func (s *Sim) Records(t time.Time, dt time.Duration, today bool) ([]flow.Record, []flow.IfCounters) {
	var recs []flow.Record
	for _, se := range s.sessions(t, dt, today) {
		c := se.c
		upPk, downPk := se.upBytes/float64(c.ipSize), se.downBytes/float64(c.ipSize)
		if c.via&viaCore != 0 {
			for dir := 0; dir < 2; dir++ {
				pk := upPk
				if dir == 1 {
					pk = downPk
				}
				if dir == 0 {
					s.link(CoreRouter, ifCoreLAN, c.coreIf, pk, float64(c.ipSize+18))
				} else {
					s.link(CoreRouter, c.coreIf, ifCoreLAN, pk, float64(c.ipSize+18))
				}
				obs := s.poisson(pk / coreRate)
				if obs == 0 {
					continue
				}
				r := flow.Record{Start: se.start, End: se.end.Add(-time.Second), Src: se.client, Dst: se.server, SrcPort: se.cport, DstPort: c.port,
					Proto: c.proto, Bytes: uint64(obs * c.ipSize), Packets: uint64(obs), Sampling: coreRate, Mult: coreRate, SamplingKnown: true,
					InIf: ifCoreLAN, OutIf: c.coreIf, Exporter: CoreRouter, Source: flow.SrcNetFlow9, Direction: 0}
				if dir == 1 {
					r.Src, r.Dst, r.SrcPort, r.DstPort, r.InIf, r.OutIf = se.server, se.client, c.port, se.cport, c.coreIf, ifCoreLAN
				}
				if c.proto == 6 {
					r.TCPFlags = 0x18
				}
				recs = append(recs, r)
			}
		}
		if c.via&(viaSwitchUp|viaSwitchSrv) != 0 {
			cp, sp := s.swPorts(&se)
			frame := float64(c.ipSize + 18)
			for dir := 0; dir < 2; dir++ {
				pk, in, out := upPk, cp, sp
				if dir == 1 {
					pk, in, out = downPk, sp, cp
				}
				s.ctrIn[in] += pk * frame
				s.ctrOut[out] += pk * frame
				s.pktIn[in] += pk
				s.pktOut[out] += pk
				obs := s.poisson(pk / switchRate)
				if obs == 0 {
					continue
				}
				r := flow.Record{Start: se.end.Add(-time.Second), End: se.end.Add(-time.Second), Src: se.client, Dst: se.server, SrcPort: se.cport, DstPort: c.port,
					Proto: c.proto, Bytes: uint64(float64(obs) * frame), Packets: uint64(obs), L2: true, Sampling: switchRate, Mult: switchRate,
					SamplingKnown: true, InIf: in, OutIf: out, Exporter: Switch, Source: flow.SrcSFlow, Observation: in, Direction: 255}
				if dir == 1 {
					r.Src, r.Dst, r.SrcPort, r.DstPort = se.server, se.client, c.port, se.cport
				}
				recs = append(recs, r)
			}
		}
		if c.via&viaFirewall != 0 {
			for dir := 0; dir < 2; dir++ {
				b := se.upBytes
				if dir == 1 {
					b = se.downBytes
				}
				pk := math.Max(1, math.Round(b/float64(c.ipSize)))
				if dir == 0 {
					s.link(Firewall, ifFwOutside, ifFwDMZ, pk, b/pk+18)
				} else if c.name != "scan" {
					s.link(Firewall, ifFwDMZ, ifFwOutside, pk, b/pk+18)
				}
				r := flow.Record{Start: se.start, End: se.end.Add(-time.Second), Src: se.client, Dst: se.server, SrcPort: se.cport, DstPort: c.port,
					Proto: c.proto, Bytes: uint64(b), Packets: uint64(pk), Sampling: 1, Mult: 1, SamplingKnown: true,
					InIf: ifFwOutside, OutIf: ifFwDMZ, Exporter: Firewall, Source: flow.SrcIPFIX, Direction: 0}
				if dir == 1 {
					r.Src, r.Dst, r.SrcPort, r.DstPort, r.InIf, r.OutIf = se.server, se.client, c.port, se.cport, ifFwDMZ, ifFwOutside
				}
				if c.name == "scan" {
					r.TCPFlags = 0x02
					if dir == 1 {
						continue
					}
				}
				recs = append(recs, r)
			}
		}
		if c.via&viaBranch != 0 {
			for dir := 0; dir < 2; dir++ {
				pk := upPk
				if dir == 1 {
					pk = downPk
				}
				if dir == 0 {
					s.link(Branch, ifBranchLAN, ifBranchWAN, pk, float64(c.ipSize+18))
				} else {
					s.link(Branch, ifBranchWAN, ifBranchLAN, pk, float64(c.ipSize+18))
				}
				obs := s.poisson(pk / branchRate)
				if obs == 0 {
					continue
				}
				r := flow.Record{Start: se.start, End: se.end.Add(-time.Second), Src: se.client, Dst: se.server, SrcPort: se.cport, DstPort: c.port,
					Proto: c.proto, Bytes: uint64(obs * c.ipSize), Packets: uint64(obs), Sampling: branchRate, Mult: branchRate, SamplingKnown: true,
					InIf: ifBranchLAN, OutIf: ifBranchWAN, Exporter: Branch, Source: flow.SrcNetFlow5}
				if dir == 1 {
					r.Src, r.Dst, r.SrcPort, r.DstPort, r.InIf, r.OutIf = se.server, se.client, c.port, se.cport, ifBranchWAN, ifBranchLAN
				}
				recs = append(recs, r)
			}
		}
	}
	return recs, s.counters(t.Add(dt))
}

// counters returns the switch counters (carried in sFlow) followed by the
// other devices' counters (read over SNMP in live mode).
func (s *Sim) counters(at time.Time) []flow.IfCounters {
	out := s.switchCounters(at)
	for _, d := range []netip.Addr{CoreRouter, Firewall, Branch} {
		out = append(out, s.DeviceCounters(at, d)...)
	}
	return out
}

func (s *Sim) switchCounters(at time.Time) []flow.IfCounters {
	var out []flow.IfCounters
	for _, idx := range []uint32{ifSwAccess, ifSwUplink, ifSwServers} {
		out = append(out, flow.IfCounters{Time: at, Exporter: Switch, IfIndex: idx, Speed: 10e9,
			InOctets: uint64(s.ctrIn[idx]), OutOctets: uint64(s.ctrOut[idx]), InPkts: uint64(s.pktIn[idx]), OutPkts: uint64(s.pktOut[idx])})
	}
	return out
}

// Backfill feeds [from, to) at one-minute steps; lastWeek marks days that
// are not "today" (the incident and this week's growth are absent).
func (s *Sim) Backfill(from, to time.Time, today bool, emit func([]flow.Record, []flow.IfCounters)) {
	for t := from.Truncate(time.Minute); t.Before(to); t = t.Add(time.Minute) {
		recs, ctrs := s.Records(t, time.Minute, today)
		emit(recs, ctrs)
	}
}

// ------------------------------------------------------------------ live

// Deliver is called with each exported datagram and its exporter address.
type Deliver func(b []byte, from netip.Addr)

// Live runs the exporters in real time, sending encoded datagrams.
// Options for the firewall's sampling rate are held back for optionsDelay
// so the "waiting for sampling rate" state can be seen.
func (s *Sim) Live(ctx context.Context, deliver Deliver, optionsDelay time.Duration) {
	start := time.Now()
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	var nf9Seq, nf5Seq, ipfixSeq, sfSeq uint32
	lastTmpl := time.Time{}
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tick.C:
			recs, ctrs := s.Records(now.Add(-5*time.Second), 5*time.Second, true)
			uptime := uint32(now.Sub(start).Milliseconds()) + 1000
			sendTmpl := now.Sub(lastTmpl) >= 60*time.Second
			if sendTmpl {
				lastTmpl = now
			}
			var core, fw, br []flow.Record
			var sw []flow.Record
			for _, r := range recs {
				switch r.Exporter {
				case CoreRouter:
					core = append(core, r)
				case Firewall:
					fw = append(fw, r)
				case Branch:
					br = append(br, r)
				case Switch:
					sw = append(sw, r)
				}
			}
			// NetFlow v9 core router, ~1.5% of export packets lost.
			for i := 0; i < len(core) || (sendTmpl && i == 0); i += 24 {
				m := encode.NewV9()
				if i == 0 && sendTmpl {
					m.AddTemplate(v9DataTmpl)
					m.AddTemplate(v9OptTmpl)
					m.AddData(v9OptTmpl.ID, []encode.Values{{encode.U(0, 4), encode.U(1, 1), encode.U(2, 1), encode.U(coreRate, 4)}})
				}
				var vals []encode.Values
				for _, r := range core[i:min(i+24, len(core))] {
					vals = append(vals, v9Values(r, now, uptime))
				}
				if len(vals) > 0 {
					m.AddData(v9DataTmpl.ID, vals)
				}
				nf9Seq++
				if s.rng.Float64() < 0.015 && !(i == 0 && sendTmpl) {
					continue
				}
				deliver(m.V9(uptime, uint32(now.Unix()), nf9Seq, 1), CoreRouter)
			}
			// IPFIX firewall, biflow records.
			pairs := biflows(fw)
			for i := 0; i < len(pairs) || (sendTmpl && i == 0); i += 20 {
				m := encode.NewIPFIX()
				if i == 0 && sendTmpl {
					m.AddTemplate(ipfixDataTmpl)
					if now.Sub(start) >= optionsDelay {
						m.AddTemplate(ipfixOptTmpl)
						m.AddData(ipfixOptTmpl.ID, []encode.Values{{encode.U(7, 4), encode.U(1, 4), encode.U(0, 4)}})
					}
				}
				var vals []encode.Values
				for _, p := range pairs[i:min(i+20, len(pairs))] {
					vals = append(vals, p)
				}
				if len(vals) > 0 {
					m.AddData(ipfixDataTmpl.ID, vals)
				}
				deliver(m.IPFIX(uint32(now.Unix()), ipfixSeq, 7), Firewall)
				ipfixSeq += uint32(m.DataRecords())
			}
			// NetFlow v5 branch router.
			for i := 0; i < len(br); i += 30 {
				var vr []encode.V5Record
				for _, r := range br[i:min(i+30, len(br))] {
					vr = append(vr, encode.V5Record{Src: r.Src, Dst: r.Dst, In: uint16(r.InIf), Out: uint16(r.OutIf), Packets: uint32(r.Packets),
						Bytes: uint32(r.Bytes), First: uptime - 5000, Last: uptime - 1000, SrcPort: r.SrcPort, DstPort: r.DstPort, Proto: r.Proto, TCPFlags: r.TCPFlags})
				}
				deliver(encode.V5(uptime, uint32(now.Unix()), nf5Seq, branchRate, vr), Branch)
				nf5Seq += uint32(len(vr))
			}
			// sFlow switch: one datagram per 8 samples; counters every tick.
			var samples []encode.SFlowSample
			for _, r := range sw {
				for k := uint64(0); k < r.Packets; k++ {
					idx := r.InIf
					s.sampleSeq[idx]++
					s.pool[idx] += switchRate
					frame := uint32(r.Bytes / r.Packets)
					hdr, _ := encode.Frame(r.Src, r.Dst, r.SrcPort, r.DstPort, r.Proto, 0, int(frame)-18, r.TCPFlags)
					samples = append(samples, encode.SFlowSample{Seq: s.sampleSeq[idx], SourceID: idx, Rate: switchRate, Pool: s.pool[idx],
						In: r.InIf, Out: r.OutIf, FrameLen: frame, Header: hdr})
				}
			}
			var cs []encode.SFlowCounter
			for _, c := range ctrs {
				if c.Exporter != Switch {
					continue
				}
				cs = append(cs, encode.SFlowCounter{IfIndex: c.IfIndex, Speed: c.Speed, InOctets: c.InOctets, OutOctets: c.OutOctets,
					InPkts: uint32(c.InPkts), OutPkts: uint32(c.OutPkts)})
			}
			for i := 0; i < len(samples) || i == 0; i += 8 {
				sfSeq++
				var cc []encode.SFlowCounter
				if i == 0 {
					cc = cs
				}
				chunk := samples[min(i, len(samples)):min(i+8, len(samples))]
				if s.rng.Float64() < 0.01 && len(cc) == 0 {
					continue // lost datagram
				}
				deliver(encode.SFlow(Switch, 0, sfSeq, uptime, chunk, cc), Switch)
			}
		}
	}
}

var v9DataTmpl = encode.Template{ID: 260, Fields: []encode.Field{
	{ID: 8, Len: 4}, {ID: 12, Len: 4}, {ID: 7, Len: 2}, {ID: 11, Len: 2}, {ID: 4, Len: 1}, {ID: 6, Len: 1},
	{ID: 1, Len: 4}, {ID: 2, Len: 4}, {ID: 22, Len: 4}, {ID: 21, Len: 4}, {ID: 10, Len: 2}, {ID: 14, Len: 2}, {ID: 48, Len: 1}, {ID: 61, Len: 1},
}}

var v9OptTmpl = encode.Template{ID: 261, Scope: []encode.Field{{ID: 1, Len: 4}},
	Fields: []encode.Field{{ID: 48, Len: 1}, {ID: 49, Len: 1}, {ID: 50, Len: 4}}}

func v9Values(r flow.Record, now time.Time, uptime uint32) encode.Values {
	first := uptime - uint32(now.Sub(r.Start).Milliseconds())
	last := uptime - uint32(now.Sub(r.End).Milliseconds())
	return encode.Values{encode.A(r.Src), encode.A(r.Dst), encode.U(uint64(r.SrcPort), 2), encode.U(uint64(r.DstPort), 2),
		encode.U(uint64(r.Proto), 1), encode.U(uint64(r.TCPFlags), 1), encode.U(r.Bytes, 4), encode.U(r.Packets, 4),
		encode.U(uint64(first), 4), encode.U(uint64(last), 4), encode.U(uint64(r.InIf), 2), encode.U(uint64(r.OutIf), 2),
		encode.U(1, 1), encode.U(0, 1)}
}

var ipfixDataTmpl = encode.Template{ID: 400, Fields: []encode.Field{
	{ID: 8, Len: 4}, {ID: 12, Len: 4}, {ID: 7, Len: 2}, {ID: 11, Len: 2}, {ID: 4, Len: 1}, {ID: 6, Len: 1},
	{ID: 1, Len: 8}, {ID: 2, Len: 8}, {ID: 1, Len: 8, Ent: 29305}, {ID: 2, Len: 8, Ent: 29305},
	{ID: 152, Len: 8}, {ID: 153, Len: 8}, {ID: 10, Len: 4}, {ID: 14, Len: 4},
}}

var ipfixOptTmpl = encode.Template{ID: 401, Scope: []encode.Field{{ID: 149, Len: 4}},
	Fields: []encode.Field{{ID: 305, Len: 4}, {ID: 306, Len: 4}}}

// biflows pairs forward and reverse firewall records into RFC 5103 records.
func biflows(recs []flow.Record) []encode.Values {
	type k struct {
		a, b   netip.Addr
		ap, bp uint16
	}
	rev := map[k]flow.Record{}
	for _, r := range recs {
		if r.InIf == ifFwDMZ {
			rev[k{r.Dst, r.Src, r.DstPort, r.SrcPort}] = r
		}
	}
	var out []encode.Values
	for _, r := range recs {
		if r.InIf != ifFwOutside {
			continue
		}
		rv := rev[k{r.Src, r.Dst, r.SrcPort, r.DstPort}]
		out = append(out, encode.Values{encode.A(r.Src), encode.A(r.Dst), encode.U(uint64(r.SrcPort), 2), encode.U(uint64(r.DstPort), 2),
			encode.U(uint64(r.Proto), 1), encode.U(uint64(r.TCPFlags), 1), encode.U(r.Bytes, 8), encode.U(r.Packets, 8),
			encode.U(rv.Bytes, 8), encode.U(rv.Packets, 8), encode.U(uint64(r.Start.UnixMilli()), 8), encode.U(uint64(r.End.UnixMilli()), 8),
			encode.U(uint64(r.InIf), 4), encode.U(uint64(r.OutIf), 4)})
	}
	return out
}
