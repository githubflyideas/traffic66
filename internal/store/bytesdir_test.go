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
