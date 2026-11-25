package store

import (
	"testing"
	"time"
)

// The interface scope is the subtraction gesture at the query layer: a
// row counts when it crossed any listed interface, in either direction
// (the traversal sense the SNMP counters use), and a negated scope
// counts everything except the listed interfaces. Rows with no
// interface at all form their own bucket, exporter/0, which can be
// subtracted like any other.
func TestIfaceScopeTotals(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	base := time.Now().UTC().Truncate(time.Minute)
	mk := func(exp string, in, out uint32, wire uint64) Row {
		return Row{TS: base, Src: "192.0.2.1", Dst: "198.51.100.1", Client: "192.0.2.1", Server: "198.51.100.1",
			App: "HTTPS", Exporter: exp, InIf: in, OutIf: out, Bytes: wire, Wire: wire, Pkts: 1, Flows: 1}
	}
	rows := []Row{
		mk("10.0.0.1", 1, 2, 1000),
		mk("10.0.0.1", 2, 3, 2000),
		mk("10.0.0.1", 3, 1, 4000),
		mk("10.0.0.2", 1, 1, 8000),
		mk("10.0.0.1", 0, 0, 16000),
	}
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	q := Query{From: base.Add(-time.Hour), To: base.Add(time.Hour)}
	wire := func(filters ...Filter) uint64 {
		qq := q
		qq.Filters = filters
		tot, err := s.Totals(qq)
		if err != nil {
			t.Fatal(err)
		}
		return tot.Wire
	}
	if got := wire(); got != 31000 {
		t.Fatalf("unscoped wire = %d, want 31000", got)
	}
	if got := wire(Filter{Field: "ifscope", Value: "10.0.0.1/1"}); got != 5000 {
		t.Fatalf("scope one interface wire = %d, want 5000", got)
	}
	if got := wire(Filter{Field: "ifscope", Value: "10.0.0.1/1,10.0.0.1/2"}); got != 7000 {
		t.Fatalf("scope two interfaces wire = %d, want 7000", got)
	}
	if got := wire(Filter{Field: "ifscope", Value: "10.0.0.1/1", Neg: true}); got != 26000 {
		t.Fatalf("subtract one interface wire = %d, want 26000", got)
	}
	if got := wire(Filter{Field: "ifscope", Value: "10.0.0.1/0", Neg: true}); got != 15000 {
		t.Fatalf("subtract unknown bucket wire = %d, want 15000", got)
	}
	if _, err := s.Totals(Query{From: q.From, To: q.To, Filters: []Filter{{Field: "ifscope", Value: "bogus"}}}); err == nil {
		t.Fatal("bad scope value accepted")
	}
}

// Interface load is the traversal split: what came in through the set
// and what went out through it, the sums the SNMP counters keep.
func TestIfaceLoad(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	base := time.Now().UTC().Truncate(time.Minute)
	mk := func(in, out uint32, wire uint64) Row {
		return Row{TS: base, Src: "192.0.2.1", Dst: "198.51.100.1", Client: "192.0.2.1", Server: "198.51.100.1",
			App: "HTTPS", Exporter: "10.0.0.1", InIf: in, OutIf: out, Bytes: wire, Wire: wire, Pkts: 1, Flows: 1}
	}
	if err := s.AppendRows([]Row{mk(1, 2, 1000), mk(2, 3, 2000), mk(3, 1, 4000)}); err != nil {
		t.Fatal(err)
	}
	q := Query{From: base.Add(-time.Hour), To: base.Add(time.Hour),
		Filters: []Filter{{Field: "ifscope", Value: "10.0.0.1/1,10.0.0.1/2"}}}
	se, err := s.IfaceLoad(q)
	if err != nil {
		t.Fatal(err)
	}
	if se.Totals[0] != 3000 || se.Totals[1] != 5000 {
		t.Fatalf("load totals in/out = %v/%v, want 3000/5000", se.Totals[0], se.Totals[1])
	}
	if _, err := s.IfaceLoad(Query{From: q.From, To: q.To}); err == nil {
		t.Fatal("load without a scope accepted")
	}
}
