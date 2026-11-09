package store

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// A device or an interface over days is answered from its summaries, and
// the answer is the one the flow records give: sealed rows and rows still
// in the hot table, in, out and in-and-out of the same interface, and the
// device's own duplicates.
func TestViewsMatchRecords(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Now().UTC().Truncate(time.Minute)
	var rows []Row
	for m := 0; m < 12*60; m += 7 {
		ts := now.Add(-time.Duration(m) * time.Minute)
		for i := 0; i < 6; i++ {
			r := Row{TS: ts, Client: fmt.Sprintf("198.51.100.%d", i), Server: fmt.Sprintf("203.0.113.%d", m%5),
				Cli4: 0xC6336400 + uint32(i), Srv4: 0xCB007100 + uint32(m%5), SvcPort: uint16(443 + i%2), Proto: 6,
				App: []string{"HTTPS", "HTTP"}[i%2], Exporter: "10.0.0.1", InIf: uint32(1 + i%3), OutIf: uint32(1 + m%3),
				Dir: DirTransit, PeerCC: "US", PeerASN: uint32(64500 + i), PeerOrg: "Org",
				Bytes: uint64(1000 * (i + 1)), Wire: uint64(1100 * (i + 1)), Pkts: uint64(i + 1), Flows: 1, Dup: i == 5}
			if i == 4 { // an IPv6 client among the IPv4 ones
				r.Client, r.Cli4 = "2001:db8::4", 0
			}
			r.Src, r.Dst = r.Client, r.Server
			rows = append(rows, r)
		}
	}
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	// everything up to two hours ago goes into a file and the summaries
	if err := s.Seal(now.Add(-2 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	for !s.viewsReady.Load() {
		time.Sleep(10 * time.Millisecond)
	}
	// long ranges start on the hour, as the API makes them
	from, to := now.Add(-11*time.Hour).Truncate(time.Hour), now.Add(time.Minute)
	for _, f := range [][]Filter{
		{{Field: "exporter", Value: "10.0.0.1"}},
		{{Field: "iface", Value: "10.0.0.1/1"}},
		{{Field: "iface_in", Value: "10.0.0.1/2"}},
		{{Field: "iface_out", Value: "10.0.0.1/2"}},
		{{Field: "iface", Value: "10.0.0.1/3"}, {Field: "iface_in", Value: "10.0.0.1/3"}},
	} {
		q := Query{From: from, To: to, Filters: f}
		if !s.rollupOK(q, false) {
			t.Fatalf("%v: not answered from the summaries", f)
		}
		answer := func() string {
			tot, err := s.Totals(q)
			if err != nil {
				t.Fatal(err)
			}
			// (the host and peer counts are worked out differently from
			// summaries, for all traffic as well, so they are not compared)
			out := fmt.Sprintf("totals %d %d %d %d cc %d\n", tot.Bytes, tot.Wire, tot.Pkts, tot.Flows, tot.Countries)
			for _, dim := range []string{"client", "server", "app", "port", "country", "asn"} {
				top, err := s.TopN(q, dim, 10)
				if err != nil {
					t.Fatal(err)
				}
				for _, r := range top {
					out += fmt.Sprintf("%s %s %d %d\n", dim, r.Key, r.Wire, r.Pkts)
				}
			}
			for _, by := range []string{"app", "client", "service"} {
				se, err := s.SeriesBy(q, by, "wire", 8)
				if err != nil {
					t.Fatal(err)
				}
				out += fmt.Sprintf("series %s %v %v\n", by, se.Names, se.Totals)
			}
			se, err := s.SeriesByApp(q, 8)
			if err != nil {
				t.Fatal(err)
			}
			var sum float64
			for _, v := range se.Values {
				for _, x := range v {
					sum += x
				}
			}
			return out + fmt.Sprintf("apps %v %.3f\n", se.Names, sum)
		}
		if f[0].Field == "exporter" {
			tot, _ := s.Totals(q)
			rs, err := s.Rings(q, "client", "server", 3, 2)
			if err != nil {
				t.Fatal(err)
			}
			var inner, outer float64
			keys := map[string]bool{}
			for _, r := range rs.Inner {
				inner += r.Wire
				keys[r.Key] = true
			}
			for _, r := range rs.Outer {
				outer += r.Wire
			}
			if rs.Total != float64(tot.Wire) || inner != rs.Total || outer != rs.Total || !keys["__other__"] || len(rs.Inner) != 4 {
				t.Errorf("rings: total %v inner %v outer %v of %d, inner %+v", rs.Total, inner, outer, tot.Wire, rs.Inner)
			}
			sv, err := s.Rings(q, "service", "client", 8, 8)
			if err != nil {
				t.Fatal(err)
			}
			seen6 := false
			for _, r := range sv.Outer {
				seen6 = seen6 || r.Key == "2001:db8::4"
			}
			if sv.Inner[0].Key != "444/tcp\tHTTP" || !seen6 {
				t.Errorf("service rings: %+v %+v", sv.Inner, sv.Outer)
			}
		}
		// equal values may come in either order
		sorted := func(x string) string { l := strings.Split(x, "\n"); sort.Strings(l); return strings.Join(l, "\n") }
		if a := answer(); f[0].Field == "exporter" && (!strings.Contains(a, "client 2001:db8::4 ") || !strings.Contains(a, " 2001:db8::4 198.")) {
			t.Errorf("%v: the IPv6 client is missing:\n%s", f, a)
		}
		got := sorted(answer())
		s.viewsReady.Store(false) // the same from the flow records
		want := sorted(answer())
		s.viewsReady.Store(true)
		if got != want {
			t.Errorf("%v:\nsummaries:\n%s\nrecords:\n%s", f, got, want)
		}
	}
}
