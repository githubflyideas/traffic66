package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestTopNGolden pins the detail-path output of TopNBy against a golden
// file captured from the implementation it was written against. Every
// any_value extra is constant within its group, so the rows themselves
// are deterministic; rows are compared as a multiset per case, because
// the order of rows tied on both the ranking measure and wire is not
// part of the contract (the queries order by those two keys and no
// more, in either implementation). The returned sequence is separately
// checked to be a valid (rank, wire desc) ordering. Regenerate with
// TOPN_GOLDEN_UPDATE=1 after an intentional change.
func TestTopNGolden(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	base := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)

	type host struct {
		text string
		num  uint32
		int_ bool
	}
	v4 := func(a, b, c, d uint32) uint32 { return a<<24 | b<<16 | c<<8 | d }
	clients := []host{}
	for i := 0; i < 10; i++ {
		clients = append(clients, host{fmt.Sprintf("198.51.100.%d", 20+i), v4(198, 51, 100, uint32(20+i)), i < 5})
	}
	for i := 0; i < 3; i++ {
		clients = append(clients, host{fmt.Sprintf("2001:db8:2::%d", 20+i), 0, i == 0})
	}
	type server struct {
		host
		cc  string
		asn uint32
		org string
		dir uint8
	}
	ccs := []string{"US", "DE", "JP", "US", "NL", "FR"}
	servers := []server{}
	for i := 0; i < 6; i++ {
		servers = append(servers, server{
			host{fmt.Sprintf("203.0.113.%d", 10+i), v4(203, 0, 113, uint32(10+i)), i == 0},
			ccs[i], uint32(64500 + i), fmt.Sprintf("Org%d", i), uint8(1 + i%4)})
	}
	servers = append(servers,
		server{host{"2001:db8:1::10", 0, false}, "US", 64600, "OrgV6A", DirOutbound},
		server{host{"2001:db8:1::11", 0, true}, "JP", 64601, "OrgV6B", DirInbound})
	ports := []struct {
		port  uint16
		proto uint8
		app   string
	}{{443, 6, "HTTPS"}, {80, 6, "HTTP"}, {53, 17, "DNS"}, {25, 6, "SMTP"}}

	var rows []Row
	c := 0
	for ci, cl := range clients {
		for si, sv := range servers {
			if (ci+si)%3 == 0 {
				continue
			}
			p := ports[(ci+si)%len(ports)]
			threat := ""
			switch {
			case c%11 == 0:
				threat = "scanner"
			case c%13 == 0:
				threat = "c2"
			}
			for m := 0; m < 2; m++ {
				wire := uint64((c+1)*7919 + m*104729 + 13)
				exp := "10.0.0.1"
				seg := "core"
				if (c+m)%2 == 1 {
					exp, seg = "10.0.0.2", "edge"
				}
				rows = append(rows, Row{
					TS:  base.Add(time.Duration(c%120+m*45) * time.Minute),
					Src: cl.text, Dst: sv.text, Client: cl.text, Server: sv.text,
					Cli4: cl.num, Srv4: sv.num, SvcPort: p.port, Proto: p.proto, App: p.app,
					Bytes: wire * 3 / 4, Wire: wire, Pkts: uint64(50*(c+1) + m*3 + 1), Flows: uint32(c%5 + 1),
					Exporter: exp, InIf: uint32(1 + c%3), OutIf: uint32(1 + (c+1)%3),
					VLAN: uint16(100 + c%3*10), Encap: uint8(c % 2), Dir: sv.dir,
					PeerCC: sv.cc, PeerASN: sv.asn, PeerOrg: sv.org, Segment: seg,
					Threat: threat, Dup: c%17 == 0, CliInt: cl.int_, SrvInt: sv.int_,
				})
			}
			c++
		}
	}
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	// seal the older half into a Parquet file, so Source is the union
	if err := s.Seal(base.Add(2*time.Hour + 30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for !s.viewsReady.Load() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	full := Query{From: base, To: base.Add(3 * time.Hour)}
	type tc struct {
		name  string
		q     Query
		dim   string
		by    string
		asc   bool
		limit int
	}
	var cases []tc
	allDims := []string{"client", "server", "service", "conv", "app", "port", "country", "asn", "proto", "dir", "exporter", "segment", "encap", "threat", "vlan"}
	for _, dim := range allDims {
		cases = append(cases,
			tc{dim + "/wire", full, dim, "wire", false, 100},
			tc{dim + "/wire-asc", full, dim, "wire", true, 100})
	}
	for _, dim := range []string{"client", "server", "service", "conv", "app", "port", "country", "asn", "threat"} {
		for _, by := range []string{"bytes", "pkts", "flows", "avg", "peers"} {
			cases = append(cases, tc{dim + "/" + by, full, dim, by, false, 100})
		}
		cases = append(cases,
			tc{dim + "/avg-asc", full, dim, "avg", true, 100},
			tc{dim + "/peers-asc", full, dim, "peers", true, 100},
			tc{dim + "/flows-asc", full, dim, "flows", true, 100})
	}
	sub := Query{From: base.Add(30 * time.Minute), To: base.Add(90 * time.Minute)}
	cases = append(cases,
		tc{"subrange/client", sub, "client", "wire", false, 100},
		tc{"subrange/conv", sub, "conv", "wire", false, 100},
		tc{"limit3/client", full, "client", "wire", false, 3},
		tc{"limit5/conv", full, "conv", "wire", false, 5},
		tc{"limit1/server-peers", full, "server", "peers", false, 1},
	)
	withFilters := func(name string, fs []Filter, dim, by string) {
		q := full
		q.Filters = fs
		cases = append(cases, tc{name, q, dim, by, false, 100})
	}
	withFilters("filter/app", []Filter{{Field: "app", Value: "HTTPS"}}, "client", "wire")
	withFilters("filter/app-neg", []Filter{{Field: "app", Value: "HTTP", Neg: true}}, "client", "wire")
	withFilters("filter/country", []Filter{{Field: "country", Value: "US"}}, "conv", "wire")
	withFilters("filter/iface", []Filter{{Field: "iface", Value: "10.0.0.1/1"}}, "client", "wire")
	withFilters("filter/iface-conv", []Filter{{Field: "iface", Value: "10.0.0.1/1"}}, "conv", "peers")
	withFilters("filter/exporter-dup", []Filter{{Field: "exporter", Value: "10.0.0.1"}}, "server", "wire")
	withFilters("filter/ip", []Filter{{Field: "ip", Value: "198.51.100.20"}}, "conv", "wire")
	withFilters("filter/port", []Filter{{Field: "port", Value: "443/tcp"}}, "conv", "wire")
	withFilters("filter/threat", []Filter{{Field: "threat", Value: "scanner"}}, "threat", "wire")
	dupq := full
	dupq.IncludeDup = true
	cases = append(cases, tc{"includedup/client", dupq, "client", "wire", false, 100})
	cases = append(cases,
		tc{"err/dim", full, "bogus", "wire", false, 10},
		tc{"err/by", full, "client", "bogus", false, 10},
		tc{"err/filter", Query{From: base, To: base.Add(time.Hour), Filters: []Filter{{Field: "bogus", Value: "x"}}}, "client", "wire", false, 10},
	)

	// rankOf is the value a case's rows are ordered by, worked out from
	// the row itself so the returned order can be checked against it.
	rankOf := func(by string, r TopRow) float64 {
		switch by {
		case "bytes":
			return float64(r.Bytes)
		case "pkts":
			return float64(r.Pkts)
		case "flows":
			return float64(r.Flows)
		case "peers":
			return float64(r.Peers)
		case "avg":
			return float64(r.Wire) / float64(max(r.Pkts, 1))
		}
		return float64(r.Wire)
	}
	var b strings.Builder
	for _, c := range cases {
		fmt.Fprintf(&b, "== %s dim=%s by=%s asc=%v limit=%d\n", c.name, c.dim, c.by, c.asc, c.limit)
		out, err := s.TopNBy(c.q, c.dim, c.limit, c.by, c.asc)
		if err != nil {
			fmt.Fprintf(&b, "ERR %v\n", err)
			continue
		}
		// the sequence must be a valid (rank, wire desc) ordering: rank
		// moves only one way, and rows tied on rank fall by wire
		for i := 1; i < len(out); i++ {
			pr, cr := rankOf(c.by, out[i-1]), rankOf(c.by, out[i])
			moved := cr > pr+1e-9
			if c.asc {
				moved = cr < pr-1e-9
			}
			if moved || (cr == pr && out[i].Wire > out[i-1].Wire) {
				t.Fatalf("%s: rows out of order at %d: rank %v wire %d after rank %v wire %d",
					c.name, i, cr, out[i].Wire, pr, out[i-1].Wire)
			}
		}
		lines := make([]string, 0, len(out))
		for _, r := range out {
			lines = append(lines, fmt.Sprintf("%q|%q|%q|%q|%d|%d|%d|%d|%d|%v",
				r.Key, r.Key2, r.Key3, r.Extra, r.Bytes, r.Wire, r.Pkts, r.Flows, r.Peers, r.Int))
		}
		// rows tied on both ordering keys have no defined order (in
		// either implementation), so a case's rows are compared as a
		// multiset: sorted, they must match the golden file exactly
		sort.Strings(lines)
		for _, l := range lines {
			b.WriteString(l + "\n")
		}
	}
	golden := filepath.Join("testdata", "topn_golden.txt")
	if os.Getenv("TOPN_GOLDEN_UPDATE") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("golden updated: %d bytes", b.Len())
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (regenerate with TOPN_GOLDEN_UPDATE=1): %v", err)
	}
	// A checkout may have turned the golden file's line endings into
	// CRLF (git's default on Windows); the generated output is LF-only,
	// so normalise before comparing.
	wantText := strings.ReplaceAll(string(want), "\r\n", "\n")
	if b.String() != wantText {
		gl, al := strings.Split(wantText, "\n"), strings.Split(b.String(), "\n")
		for i := 0; i < len(gl) || i < len(al); i++ {
			var g, a string
			if i < len(gl) {
				g = gl[i]
			}
			if i < len(al) {
				a = al[i]
			}
			if g != a {
				t.Fatalf("first difference at line %d:\ngolden: %s\nactual: %s", i+1, g, a)
			}
		}
		t.Fatal("golden mismatch (length only)")
	}
}
