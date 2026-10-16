package sandbox

import (
	"bufio"
	"encoding/binary"
	"io"
	"math/rand/v2"
	"net/netip"
	"sort"
	"time"

	"github.com/githubflyideas/traffic66/internal/pcapfile"
)

// SampleName is the file name of the demo's example capture.
const SampleName = "example-office-attack.pcap"

// Sample writes the demo's example capture: 15 minutes of an office network
// ending at end, with an attack inside it. A compromised PC (10.10.3.27)
// scans every low port of the file server, sweeps the office for SMB, then
// guesses RDP passwords on the file server. Only headers are captured (snap length 54),
// so the file stays small while the traffic sizes are real.
func Sample(w io.Writer, end time.Time) error {
	rng := rand.New(rand.NewPCG(66, 6))
	start := end.Add(-15 * time.Minute)
	type pkt struct {
		t            time.Time
		src, dst     netip.Addr
		sport, dport uint16
		proto, flags uint8
		size         int
	}
	var ps []pkt
	at := func(from, d time.Duration) time.Time {
		return start.Add(from + time.Duration(rng.Int64N(int64(d))))
	}
	ip := func(s string) netip.Addr { return netip.MustParseAddr(s) }
	office := func() netip.Addr {
		return netip.AddrFrom4([4]byte{10, 10, byte(1 + rng.IntN(4)), byte(2 + rng.IntN(250))})
	}
	// ordinary traffic: web, video calls, DNS, file shares
	servers := []struct {
		a     string
		port  uint16
		proto uint8
		mbit  float64
	}{
		{"142.250.196.110", 443, 6, 6}, {"151.101.1.140", 443, 6, 3}, {"13.107.42.14", 443, 6, 4},
		{"104.16.132.229", 443, 6, 2}, {"52.84.150.39", 443, 6, 3}, {"170.114.52.2", 8801, 17, 2.5},
		{"114.114.114.114", 53, 17, 0.05}, {"8.8.8.8", 53, 17, 0.05}, {"10.20.0.15", 445, 6, 5},
	}
	clients := make([]netip.Addr, 40)
	for i := range clients {
		clients[i] = office()
	}
	for _, s := range servers {
		srv := ip(s.a)
		n := int(s.mbit * 1e6 / 8 / 1200 * 900 / 50) // packets in 15 minutes, at 1/50 of the rate
		for i := 0; i < n; i++ {
			c := clients[rng.IntN(len(clients))]
			// one session per client and server
			sport := uint16(49152 + (int(c.As4()[2])*251+int(c.As4()[3])+int(s.port))%16000)
			size := 1200 + rng.IntN(300)
			if s.port == 53 {
				size = 80 + rng.IntN(60)
			}
			t := at(0, 15*time.Minute)
			if rng.IntN(5) == 0 { // requests and acks upstream
				ps = append(ps, pkt{t, c, srv, sport, s.port, s.proto, 0x18, 66 + rng.IntN(400)})
			} else {
				ps = append(ps, pkt{t, srv, c, s.port, sport, s.proto, 0x18, size})
			}
		}
	}
	pc, fileServer := ip("10.10.3.27"), ip("10.20.0.15")
	// minute 3-5: SMB sweep of the office, one SYN per address
	for i := 0; i < 600; i++ {
		ps = append(ps, pkt{at(3*time.Minute, 2*time.Minute), pc, netip.AddrFrom4([4]byte{10, 10, byte(1 + i/250), byte(2 + i%250)}),
			uint16(40000 + rng.IntN(20000)), 445, 6, 0x02, 60})
	}
	// minute 1-2: every port up to 1024 on the file server
	for p := 1; p <= 1024; p++ {
		ps = append(ps, pkt{at(time.Minute, time.Minute), pc, fileServer, 51000, uint16(p), 6, 0x02, 60})
	}
	// minute 12-15: RDP password guessing, 300 short connections
	for i := 0; i < 300; i++ {
		t := at(12*time.Minute, 3*time.Minute-200*time.Millisecond)
		sport := uint16(50000 + i)
		for j := 0; j < 8; j++ {
			ps = append(ps, pkt{t.Add(time.Duration(j) * 20 * time.Millisecond), pc, fileServer, sport, 3389, 6, 0x18, 90 + rng.IntN(150)})
		}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].t.Before(ps[j].t) })

	bw := bufio.NewWriter(w)
	pw, err := pcapfile.NewWriter(bw, pcapfile.LinkEthernet)
	if err != nil {
		return err
	}
	buf := make([]byte, 54)
	for _, p := range ps {
		n := frame(buf, p.src, p.dst, p.sport, p.dport, p.proto, p.flags, p.size)
		if err := pw.Write(p.t, buf[:n], p.size); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// frame writes Ethernet, IPv4 and TCP/UDP headers for a packet of size
// bytes on the wire and returns how many header bytes were written.
func frame(b []byte, src, dst netip.Addr, sport, dport uint16, proto, flags uint8, size int) int {
	clear(b)
	copy(b[0:6], []byte{0, 0x1b, 0x21, 1, 2, 3})
	copy(b[6:12], []byte{0, 0x1b, 0x21, 4, 5, 6})
	binary.BigEndian.PutUint16(b[12:], 0x0800)
	ipb := b[14:]
	ipb[0] = 0x45
	binary.BigEndian.PutUint16(ipb[2:], uint16(max(size-14, 40)))
	ipb[8] = 64
	ipb[9] = proto
	s4, d4 := src.As4(), dst.As4()
	copy(ipb[12:16], s4[:])
	copy(ipb[16:20], d4[:])
	l4 := ipb[20:]
	binary.BigEndian.PutUint16(l4[0:], sport)
	binary.BigEndian.PutUint16(l4[2:], dport)
	if proto == 17 {
		binary.BigEndian.PutUint16(l4[4:], uint16(max(size-34, 8)))
		return 14 + 20 + 8
	}
	l4[12] = 5 << 4
	l4[13] = flags
	return 14 + 20 + 20
}
