package store

import (
	"strconv"
	"testing"
	"time"
)

// A device sampling one interface (in and out): the other interfaces of its
// flows are peers. A device saying which interface sampled each flow: only
// those are sampled.
func TestPeers(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ts := time.Now().UTC().Truncate(time.Hour).Add(-30 * time.Minute)
	row := func(exp string, in, out uint32, w uint64) Row {
		return Row{TS: ts, Exporter: exp, InIf: in, OutIf: out, Wire: w, Bytes: w, Pkts: 1, Flows: 1}
	}
	rows := []Row{
		row("10.0.0.5", 10, 34, 500), row("10.0.0.5", 10, 30, 300), row("10.0.0.5", 26, 10, 200), row("10.0.0.5", 22, 10, 100),
		row("10.0.0.6", 1, 2, 400), row("10.0.0.6", 2, 3, 400), row("10.0.0.6", 3, 1, 400),
	}
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	h := ts.Truncate(time.Hour)
	dim := map[DimKey]*Counters{
		{TS: h, Dim: "obsunk", Val: "10.0.0.5"}: {Wire: 1100},
		{TS: h, Dim: "obsif", Val: "10.0.0.6/1"}: {Wire: 400},
		{TS: h, Dim: "obsif", Val: "10.0.0.6/2"}: {Wire: 400},
	}
	if err := s.AppendRollups(nil, nil, dim); err != nil {
		t.Fatal(err)
	}
	list, err := s.Interfaces(Query{From: ts.Add(-time.Hour), To: ts.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"10.0.0.5/10": false, "10.0.0.5/34": true, "10.0.0.5/30": true, "10.0.0.5/26": true, "10.0.0.5/22": true,
		"10.0.0.6/1": false, "10.0.0.6/2": false, "10.0.0.6/3": true}
	if len(list) != len(want) {
		t.Fatalf("%d interfaces, want %d", len(list), len(want))
	}
	for _, f := range list {
		k := f.Exporter + "/" + strconv.FormatUint(uint64(f.IfIndex), 10)
		if f.Peer != want[k] {
			t.Errorf("%s peer=%v", k, f.Peer)
		}
	}
	if list[len(list)-1].Peer != true || list[0].Peer {
		t.Error("peers not last")
	}
}
