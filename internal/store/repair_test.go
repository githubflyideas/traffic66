package store

import (
	"testing"
	"time"
)

// Counters that wrapped below zero (written by 1.6.0 and earlier) are zeroed
// in the database and in the flow files when the store opens, so sums work.
func TestRepairWrapped(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(Options{Dir: dir, Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	old := now.Add(-3 * time.Hour).Truncate(time.Minute)
	recent := now.Truncate(time.Minute)
	wrapped := ^uint64(0) - 76 // -77 as unsigned
	row := func(ts time.Time, b uint64) Row {
		return Row{TS: ts, Src: "10.0.0.1", Dst: "10.0.0.2", Client: "10.0.0.1", Server: "10.0.0.2", Exporter: "127.0.0.1",
			Bytes: b, Wire: b, Pkts: 1, Flows: 1}
	}
	if err := s.AppendRows([]Row{row(old, 1000), row(old, wrapped), row(recent, 500), row(recent, wrapped)}); err != nil {
		t.Fatal(err)
	}
	if err := s.Seal(now); err != nil { // the old rows go to a Parquet file
		t.Fatal(err)
	}
	h := old.Truncate(time.Hour)
	if err := s.AppendRollups(map[TSKey]*Counters{{TS: h, App: "x"}: {Bytes: wrapped, Wire: wrapped, Pkts: 1}}, nil,
		map[DimKey]*Counters{{TS: h, Dim: "obsunk", Val: "127.0.0.1"}: {Wire: wrapped}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(`DELETE FROM meta WHERE k = 'repair_wrapped_v1'`); err != nil { // as if written by an old version
		t.Fatal(err)
	}
	s.Close()

	s, err = Open(Options{Dir: dir, Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var b, w uint64
	if err := s.DB.QueryRow(`SELECT sum(bytes)::UBIGINT, sum(wire)::UBIGINT FROM ` + s.Source(old.Add(-time.Hour), now.Add(time.Hour))).Scan(&b, &w); err != nil {
		t.Fatal(err)
	}
	if b != 1500 || w != 1500 {
		t.Fatalf("flows bytes %d wire %d", b, w)
	}
	for _, tb := range []string{"r_ts", "r_dim"} {
		var n int
		s.DB.QueryRow(`SELECT count(*) FROM ` + tb + ` WHERE wire > 4611686018427387904`).Scan(&n)
		if n != 0 {
			t.Fatalf("%s still has %d wrapped rows", tb, n)
		}
	}
	if s.Meta("repair_wrapped_v1") == "" {
		t.Fatal("repair not recorded")
	}
}
