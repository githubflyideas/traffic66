package store

import (
	"testing"
	"time"
)

// An office that only opens connections outwards still has inbound
// traffic: the replies it downloads. The byte-direction filter counts
// them; the connection-direction filter (dir=inbound) rightly does not.
func TestBytesDirection(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ts := time.Now().UTC().Add(-10 * time.Minute)
	pc, web := "10.1.1.5", "203.0.113.9"
	base := Row{TS: ts, Client: pc, Server: web, Cli4: 0x0A010105, Srv4: 0xCB007109, Dir: DirOutbound, CliInt: true, Pkts: 1, Flows: 1, Exporter: "10.0.0.1"}
	req, resp := base, base
	req.Src, req.Dst, req.Wire, req.Bytes = pc, web, 1000, 1000     // the request, out
	resp.Src, resp.Dst, resp.Wire, resp.Bytes = web, pc, 9000, 9000 // the download, in
	if err := s.AppendRows([]Row{req, resp}); err != nil {
		t.Fatal(err)
	}
	q := func(f ...Filter) uint64 {
		tot, err := s.Totals(Query{From: ts.Add(-time.Minute), To: ts.Add(time.Minute), Filters: f})
		if err != nil {
			t.Fatal(err)
		}
		return tot.Wire
	}
	if got := q(Filter{Field: "bytes", Value: "in"}); got != 9000 {
		t.Errorf("bytes in = %d, want the 9000 downloaded", got)
	}
	if got := q(Filter{Field: "bytes", Value: "out"}); got != 1000 {
		t.Errorf("bytes out = %d, want the 1000 sent", got)
	}
	if got := q(Filter{Field: "dir", Value: "inbound"}); got != 0 {
		t.Errorf("dir=inbound = %d: no connection was opened from outside", got)
	}
	if _, err := s.Totals(Query{From: ts, To: ts.Add(time.Minute), Filters: []Filter{{Field: "bytes", Value: "sideways"}}}); err == nil {
		t.Error("a bad byte direction was accepted")
	}
}

// A data centre whose service addresses are public and not listed as its
// own networks: every flow is transit by address, so only the interface
// can say which way it went. In through the uplink is inbound, out
// through it is outbound, the same split as the interface's own chart.
func TestInterfaceDirection(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ts := time.Now().UTC().Add(-10 * time.Minute)
	user, vip := "198.51.100.7", "203.0.113.10"
	base := Row{TS: ts, Client: user, Server: vip, Cli4: 0xC6336407, Srv4: 0xCB00710A, Dir: DirTransit, Pkts: 1, Flows: 1, Exporter: "10.0.0.1"}
	req, resp := base, base
	req.Src, req.Dst, req.Wire, req.InIf, req.OutIf = user, vip, 1000, 7, 3      // in through the uplink 7
	resp.Src, resp.Dst, resp.Wire, resp.InIf, resp.OutIf = vip, user, 9000, 3, 7 // back out through it
	resp.Dup = true                                                              // and seen by a second device too
	if err := s.AppendRows([]Row{req, resp}); err != nil {
		t.Fatal(err)
	}
	q := func(f ...Filter) uint64 {
		tot, err := s.Totals(Query{From: ts.Add(-time.Minute), To: ts.Add(time.Minute), Filters: f})
		if err != nil {
			t.Fatal(err)
		}
		return tot.Wire
	}
	if got := q(Filter{Field: "bytes", Value: "in"}); got != 0 {
		t.Errorf("bytes in = %d: by address nothing is inside", got)
	}
	if got := q(Filter{Field: "iface_in", Value: "10.0.0.1/7"}); got != 1000 {
		t.Errorf("in through the uplink = %d, want 1000", got)
	}
	if got := q(Filter{Field: "iface_out", Value: "10.0.0.1/7"}); got != 9000 {
		t.Errorf("out through the uplink = %d, want 9000 (its own duplicate counts)", got)
	}
}
