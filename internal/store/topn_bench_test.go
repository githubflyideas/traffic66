package store

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkTopN exercises the detail path of TopNBy at production-like
// scale: 360k minute rows over four hours (the older half sealed into a
// Parquet file), 20k distinct clients (10% of them IPv6), 500 servers.
func BenchmarkTopN(b *testing.B) {
	s, err := Open(Options{Dir: b.TempDir(), Threads: 2})
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	base := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)

	type host struct {
		text string
		num  uint32
	}
	clients := make([]host, 0, 20000)
	for i := 0; i < 18000; i++ {
		n := uint32(0x0A000000 + i)
		clients = append(clients, host{fmt.Sprintf("10.%d.%d.%d", (n>>16)&255, (n>>8)&255, n&255), n})
	}
	for i := 0; i < 2000; i++ {
		clients = append(clients, host{fmt.Sprintf("2001:db8:3::%x", i), 0})
	}
	servers := make([]host, 0, 500)
	for i := 0; i < 450; i++ {
		n := uint32(0xCB007100 + i)
		servers = append(servers, host{fmt.Sprintf("203.0.113.%d", i%250+1), n})
	}
	for i := 0; i < 50; i++ {
		servers = append(servers, host{fmt.Sprintf("2001:db8:4::%x", i), 0})
	}
	ports := []struct {
		port  uint16
		proto uint8
		app   string
	}{{443, 6, "HTTPS"}, {80, 6, "HTTP"}, {53, 17, "DNS"}, {22, 6, "SSH"}}
	ccs := []string{"US", "DE", "JP", "NL", "FR", "GB"}

	seed := uint64(88172645463325252)
	next := func() uint64 { // xorshift64, deterministic
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		return seed
	}
	batch := make([]Row, 0, 30000)
	flush := func() {
		if len(batch) > 0 {
			if err := s.AppendRows(batch); err != nil {
				b.Fatal(err)
			}
			batch = batch[:0]
		}
	}
	for m := 0; m < 240; m++ {
		ts := base.Add(time.Duration(m) * time.Minute)
		for i := 0; i < 1500; i++ {
			r := next()
			cl := clients[r%20000]
			sv := servers[(r>>21)%500]
			p := ports[(r>>42)%4]
			wire := 500 + r%9000
			batch = append(batch, Row{
				TS: ts, Src: cl.text, Dst: sv.text, Client: cl.text, Server: sv.text,
				Cli4: cl.num, Srv4: sv.num, SvcPort: p.port, Proto: p.proto, App: p.app,
				Bytes: wire * 3 / 4, Wire: wire, Pkts: 1 + r%90, Flows: 1,
				Exporter: "10.0.0.1", InIf: 1 + uint32(r%4), OutIf: 1 + uint32((r>>9)%4),
				Dir: uint8(1 + r%4), PeerCC: ccs[(r>>15)%6], PeerASN: uint32(64500 + (r>>21)%500),
				PeerOrg: "Org", Segment: "core", CliInt: cl.num>>24 == 10, SrvInt: false,
			})
			if len(batch) == cap(batch) {
				flush()
			}
		}
	}
	flush()
	if err := s.Seal(base.Add(2*time.Hour + 30*time.Minute)); err != nil {
		b.Fatal(err)
	}
	q := Query{From: base, To: base.Add(4 * time.Hour)}
	run := func(name, dim, by string, asc bool) {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rows, err := s.TopNBy(q, dim, 66, by, asc)
				if err != nil {
					b.Fatal(err)
				}
				if len(rows) == 0 {
					b.Fatal("no rows")
				}
			}
		})
	}
	run("ClientWire", "client", "wire", false)
	run("ConvWire", "conv", "wire", false)
	run("ClientAvgAsc", "client", "avg", true)
	run("ServerBytes", "server", "bytes", false)
	run("ClientPeers", "client", "peers", false)
}

// BenchmarkTopNXL is the production-scale tier, sized from a real
// installation: a measured 15-minute range there holds 130k rows, so
// a one-hour range holds ~520k (and the whole raw store is 67.68M
// records over four days). The fixture holds 9.9M minute rows over
// 19.8 hours at that density (499,980 rows per hour), with 300k
// distinct clients and 3k servers; the first thirteen hours are
// sealed into two Parquet files, the rest stays hot. Every query
// takes the detail path — conv has no rollup, and the ranges are at
// or under the six-hour rollup span — so the one-hour numbers are the
// Top page's cost at production density, over ~500k scanned rows.
func BenchmarkTopNXL(b *testing.B) {
	s, err := Open(Options{Dir: b.TempDir(), Threads: 2})
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	base := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)

	type host struct {
		text string
		num  uint32
		cc   string
	}
	ccs := []string{"US", "DE", "JP", "NL", "FR", "GB"}
	clients := make([]host, 0, 300000)
	for i := 0; i < 270000; i++ {
		n := uint32(0x0A000000 + i)
		clients = append(clients, host{fmt.Sprintf("10.%d.%d.%d", (n>>16)&255, (n>>8)&255, n&255), n, ""})
	}
	for i := 0; i < 30000; i++ {
		clients = append(clients, host{fmt.Sprintf("2001:db8:3::%x", i), 0, ""})
	}
	servers := make([]host, 0, 3000)
	for i := 0; i < 2700; i++ {
		n := uint32(0xCB000000 + i)
		servers = append(servers, host{fmt.Sprintf("203.%d.%d.%d", (n>>16)&255, (n>>8)&255, n&255), n, ccs[i%len(ccs)]})
	}
	for i := 0; i < 300; i++ {
		servers = append(servers, host{fmt.Sprintf("2001:db8:4::%x", i), 0, ccs[i%len(ccs)]})
	}
	ports := []struct {
		port  uint16
		proto uint8
		app   string
	}{{443, 6, "HTTPS"}, {80, 6, "HTTP"}, {53, 17, "DNS"}, {22, 6, "SSH"},
		{25, 6, "SMTP"}, {993, 6, "IMAPS"}, {8080, 6, "HTTP"}, {1194, 17, "VPN"},
		{5060, 17, "SIP"}, {3306, 6, "MySQL"}, {5432, 6, "PostgreSQL"}, {6379, 6, "Redis"}}

	seed := uint64(1442695040888963407)
	next := func() uint64 {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		return seed
	}
	batch := make([]Row, 0, 50000)
	flush := func() {
		if len(batch) > 0 {
			if err := s.AppendRows(batch); err != nil {
				b.Fatal(err)
			}
			batch = batch[:0]
		}
	}
	const perMin = 8333 // 1188 minutes of it is 9,899,604 rows
	for m := 0; m < 1188; m++ {
		ts := base.Add(time.Duration(m) * time.Minute)
		for i := 0; i < perMin; i++ {
			r := next()
			cl := clients[r%300000]
			sv := servers[(r>>23)%3000]
			p := ports[(r>>43)%12]
			wire := 500 + r%9000
			batch = append(batch, Row{
				TS: ts, Src: cl.text, Dst: sv.text, Client: cl.text, Server: sv.text,
				Cli4: cl.num, Srv4: sv.num, SvcPort: p.port, Proto: p.proto, App: p.app,
				Bytes: wire * 3 / 4, Wire: wire, Pkts: 1 + r%90, Flows: 1,
				Exporter: "10.0.0.1", InIf: 1 + uint32(r%4), OutIf: 1 + uint32((r>>9)%4),
				Dir: uint8(1 + r%4), PeerCC: sv.cc, PeerASN: uint32(64500 + (r>>23)%3000),
				PeerOrg: "Org", Segment: "core", CliInt: cl.num>>24 == 10, SrvInt: false,
			})
			if len(batch) == cap(batch) {
				flush()
			}
		}
	}
	flush()
	for _, h := range []int{6, 13} {
		if err := s.Seal(base.Add(time.Duration(h)*time.Hour + 30*time.Minute)); err != nil {
			b.Fatal(err)
		}
	}
	hour := func(from int) Query {
		return Query{From: base.Add(time.Duration(from) * time.Hour), To: base.Add(time.Duration(from+1) * time.Hour)}
	}
	run := func(name string, q Query, dim, by string) {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rows, err := s.TopNBy(q, dim, 66, by)
				if err != nil {
					b.Fatal(err)
				}
				if len(rows) == 0 {
					b.Fatal("no rows")
				}
			}
		})
	}
	// one hour inside the sealed files, one in the hot table (~500k
	// rows each), and six hours across the seal boundary (~3.0M rows)
	run("ClientWire1hSealed", hour(2), "client", "wire")
	run("ConvWire1hSealed", hour(2), "conv", "wire")
	run("ClientWire1hHot", hour(18), "client", "wire")
	run("ConvWire6hMixed", Query{From: base.Add(10 * time.Hour), To: base.Add(16 * time.Hour)}, "conv", "wire")
}
