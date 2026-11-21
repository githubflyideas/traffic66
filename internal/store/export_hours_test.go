package store_test

import (
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/store"
)

// The rollup export adds counter snapshots up into hours: an interval's
// traffic belongs to the hours it overlaps, in proportion; an interval
// after a long gap or a counter reset counts nothing, because its size
// is unknowable — the same rule the reconciliation page applies.
func TestIfaceHours(t *testing.T) {
	st, err := store.Open(store.Options{Dir: t.TempDir(), RawDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	day := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	at := func(h, m int) time.Time { return day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute) }
	snap := func(ts time.Time, in, out, inp, outp uint64) {
		if _, err := st.DB.Exec(`INSERT INTO ifc (ts, exporter, ifindex, speed, in_oct, out_oct, in_pkts, out_pkts)
			VALUES (?, '10.0.0.1', 3, 1000000000, ?, ?, ?, ?)`, ts, in, out, inp, outp); err != nil {
			t.Fatal(err)
		}
	}
	snap(at(10, 30), 0, 0, 0, 0)
	snap(at(10, 50), 1200, 600, 12, 6)   // +1200 over 20m, inside hour 10
	snap(at(11, 10), 2400, 1200, 24, 12) // +1200 over 20m, split across 10 and 11
	snap(at(12, 30), 9999, 9999, 99, 99) // gap of 80m: the interval counts nothing
	snap(at(12, 40), 100, 50, 1, 1)      // counters fell (a reset): counts nothing
	snap(at(12, 50), 700, 350, 7, 4)     // +600 over 10m, inside hour 12

	rows, err := st.IfaceHours(store.Query{From: at(10, 0), To: at(13, 0)})
	if err != nil {
		t.Fatal(err)
	}
	want := []store.IfaceHour{
		{Start: at(10, 0), Exporter: "10.0.0.1", IfIndex: 3, InOct: 1800, OutOct: 900, InPkts: 18, OutPkts: 9},
		{Start: at(11, 0), Exporter: "10.0.0.1", IfIndex: 3, InOct: 600, OutOct: 300, InPkts: 6, OutPkts: 3},
		{Start: at(12, 0), Exporter: "10.0.0.1", IfIndex: 3, InOct: 600, OutOct: 300, InPkts: 6, OutPkts: 3},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows %v, want %d", len(rows), rows, len(want))
	}
	for i := range want {
		if rows[i] != want[i] {
			t.Fatalf("row %d = %+v, want %+v", i, rows[i], want[i])
		}
	}

	// A window that starts mid-interval clips it: only the part inside
	// the window counts.
	rows, err = st.IfaceHours(store.Query{From: at(10, 45), To: at(11, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].InOct != 900 || rows[0].OutOct != 450 {
		t.Fatalf("clipped window: %+v", rows)
	}
}
