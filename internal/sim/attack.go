package sim

import (
	"math/rand/v2"
	"net/netip"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
)

// The demo's attack, a few hours before "now", as the sampled switch and the
// unsampled firewall report it: the finance PC, already compromised, scans
// the office and server networks for SMB, scans every port of the file
// server, guesses RDP passwords on it, copies itself to six workstations over
// SMB and then uploads data to its control server (the incident in model.go).
// Later a botnet floods the public website with small UDP packets.
//
// Times are offsets before the incident start (now - 5 h).
type attackPhase struct {
	name       string
	start, end time.Duration // before incidentStart (start > end)
}

var phases = []attackPhase{
	{"sweep", 90 * time.Minute, 80 * time.Minute},
	{"portscan", 75 * time.Minute, 70 * time.Minute},
	{"rdp", 65 * time.Minute, 50 * time.Minute},
	{"smb", 40 * time.Minute, 20 * time.Minute},
}

// Workstations the finance PC copies itself to.
var lateralTargets = []netip.Addr{
	netip.MustParseAddr("10.10.1.23"), netip.MustParseAddr("10.10.1.87"), netip.MustParseAddr("10.10.2.14"),
	netip.MustParseAddr("10.10.2.66"), netip.MustParseAddr("10.10.4.9"), netip.MustParseAddr("10.10.4.131"),
}

func (m *model) phase(name string, t time.Time) bool {
	for _, p := range phases {
		if p.name == name {
			return !t.Before(m.incidentStart.Add(-p.start)) && t.Before(m.incidentStart.Add(-p.end))
		}
	}
	return false
}

// flood: two hours before now, for ten minutes.
func (m *model) flooding(t time.Time) bool {
	s := m.incidentStart.Add(3 * time.Hour)
	return !t.Before(s) && t.Before(s.Add(10*time.Minute))
}

// switchSample reports n packets of frame bytes from src to dst the way the
// sampling switch does: each packet is sampled with probability 1/rate and
// becomes one record. Interface counters see every packet.
func (s *Sim) switchSample(t time.Time, dt time.Duration, n float64, frame float64, flags uint8,
	pick func(r *rand.Rand) (src, dst netip.Addr, sport, dport uint16)) []flow.Record {
	in, out := uint32(ifSwAccess), uint32(ifSwServers)
	s.ctrIn[in] += n * frame
	s.ctrOut[out] += n * frame
	s.pktIn[in] += n
	s.pktOut[out] += n
	k := s.poisson(n / switchRate)
	recs := make([]flow.Record, 0, k)
	for i := 0; i < k; i++ {
		src, dst, sp, dp := pick(s.rng)
		at := t.Add(time.Duration(s.rng.Int64N(int64(dt))))
		recs = append(recs, flow.Record{Start: at, End: at, Src: src, Dst: dst, SrcPort: sp, DstPort: dp, Proto: 6, TCPFlags: flags,
			Bytes: uint64(frame), Packets: 1, L2: true, Sampling: switchRate, Mult: switchRate, SamplingKnown: true,
			InIf: in, OutIf: out, Exporter: Switch, Source: flow.SrcSFlow, Observation: in, Direction: 255})
	}
	return recs
}

func (s *Sim) attackRecords(t time.Time, dt time.Duration) []flow.Record {
	m := s.m
	secs := dt.Seconds()
	var recs []flow.Record
	office := pool("10.10.0.0/16", 60000)
	servers := pool("10.20.0.0/24", 240)
	if m.phase("sweep", t) {
		// SMB sweep: 2,000 SYN per second, one per address
		recs = append(recs, s.switchSample(t, dt, 2000*secs, 64, 0x02, func(r *rand.Rand) (netip.Addr, netip.Addr, uint16, uint16) {
			dst := office(r)
			if r.IntN(10) == 0 {
				dst = servers(r)
			}
			return financePC, dst, uint16(40000 + r.IntN(20000)), 445
		})...)
	}
	if m.phase("portscan", t) {
		// every TCP port of the file server, twice, in five minutes
		recs = append(recs, s.switchSample(t, dt, 131072/300.0*secs, 64, 0x02, func(r *rand.Rand) (netip.Addr, netip.Addr, uint16, uint16) {
			return financePC, fileServer, 51000, uint16(1 + r.IntN(65535))
		})...)
	}
	if m.phase("rdp", t) {
		// RDP password guessing: 10 attempts a second, ~15 small packets each
		recs = append(recs, s.switchSample(t, dt, 150*secs, 170, 0x18, func(r *rand.Rand) (netip.Addr, netip.Addr, uint16, uint16) {
			return financePC, fileServer, uint16(49152 + r.IntN(16000)), 3389
		})...)
	}
	if m.phase("smb", t) {
		// copying to six workstations over SMB, 30 Mbit/s in all
		recs = append(recs, s.switchSample(t, dt, 30e6/8/1300*secs, 1318, 0x18, func(r *rand.Rand) (netip.Addr, netip.Addr, uint16, uint16) {
			return financePC, lateralTargets[r.IntN(len(lateralTargets))], uint16(50000 + r.IntN(8)), 445
		})...)
	}
	if m.flooding(t) {
		// 60,000 packets a second of 46-byte UDP from 200 addresses, through
		// the firewall (IPFIX, unsampled)
		bots := pool("198.18.7.0/24", 200)
		per := 60000 * secs / 200
		for i := 0; i < 200; i++ {
			src := bots(s.rng)
			s.link(Firewall, ifFwOutside, ifFwDMZ, per, 64)
			recs = append(recs, flow.Record{Start: t, End: t.Add(dt - time.Second), Src: src, Dst: dmzWeb, SrcPort: uint16(1024 + s.rng.IntN(60000)), DstPort: 80,
				Proto: 17, Bytes: uint64(per * 46), Packets: uint64(per), Sampling: 1, Mult: 1, SamplingKnown: true,
				InIf: ifFwOutside, OutIf: ifFwDMZ, Exporter: Firewall, Source: flow.SrcIPFIX})
		}
	}
	return recs
}
