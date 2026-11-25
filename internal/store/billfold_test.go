package store

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// billDump is r_bill canonicalised: partial rows of one pair of
// addresses summed, so two stores that folded the same records in
// different chunks compare equal exactly when their summaries agree.
func billDump(t *testing.T, s *Store) string {
	t.Helper()
	rows, err := s.DB.Query(`SELECT ts, dup, src, dst, sum(wire) FROM r_bill GROUP BY ALL`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var ts time.Time
		var dup bool
		var src, dst string
		var w uint64
		if err := rows.Scan(&ts, &dup, &src, &dst, &w); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, fmt.Sprintf("%s|%v|%s|%s|%d", ts.Format("2006-01-02 15:04"), dup, src, dst, w))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// The billing summaries a store ends up with must not depend on how
// the work was split between the folder and the seals: folded a few
// minutes at a time as the data settled, folded by the seals alone,
// or folded in one pass over the hot table, the summaries of the
// same records are the same rows — a bill is written from them.
func TestBillFoldAndSealAgree(t *testing.T) {
	base := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	rows := foldFixture(base)
	end := base.Add(6 * time.Hour)

	one := openFoldStore(t)
	defer one.Close()
	if err := one.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	tx, err := one.DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := one.foldTx(tx, time.Time{}, end); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	want := billDump(t, one)
	if want == "" {
		t.Fatal("one fold produced no billing summaries")
	}

	sealed := openFoldStore(t)
	defer sealed.Close()
	if err := sealed.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	for _, now := range []time.Time{base.Add(2 * time.Hour), base.Add(4 * time.Hour), end} {
		if err := sealed.Seal(now); err != nil {
			t.Fatal(err)
		}
	}
	if got := billDump(t, sealed); got != want {
		t.Fatalf("seal-only summaries differ from one fold:\n%s\n---\n%s", got, want)
	}

	both := openFoldStore(t)
	defer both.Close()
	if err := both.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	for _, now := range []time.Time{base.Add(time.Hour), base.Add(2 * time.Hour)} {
		if err := both.foldOnce(now); err != nil {
			t.Fatal(err)
		}
		if err := both.foldOnce(now); err != nil { // again: a no-op
			t.Fatal(err)
		}
	}
	if err := both.Seal(base.Add(3 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := both.foldOnce(base.Add(5 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := both.Seal(end); err != nil {
		t.Fatal(err)
	}
	if got := billDump(t, both); got != want {
		t.Fatalf("fold+seal summaries differ from one fold:\n%s\n---\n%s", got, want)
	}
}

// A record that arrives after its minute was folded is counted when
// its hour is sealed, in the billing summaries as in the
// conversation ones: the seal rebuilds the hours it touches from
// the records.
func TestBillCountsLateRowsAtSeal(t *testing.T) {
	base := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	s := openFoldStore(t)
	defer s.Close()
	rows := foldFixture(base)
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	if err := s.foldOnce(base.Add(2 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	late := rows[0]
	late.Wire, late.Bytes = 7777, 5555
	wireOf := func() uint64 {
		var w uint64
		err := s.DB.QueryRow(`SELECT coalesce(sum(wire), 0) FROM r_bill
			WHERE ts = ? AND src = ? AND dst = ? AND dup = ?`,
			late.TS.Truncate(time.Hour), late.Src, late.Dst, late.Dup).Scan(&w)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	before := wireOf()
	if err := s.AppendRows([]Row{late}); err != nil {
		t.Fatal(err)
	}
	if err := s.foldOnce(base.Add(3 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := wireOf(); got != before {
		t.Fatalf("the fold counted a late record: wire %d, want %d (it is counted at the seal)", got, before)
	}
	if err := s.Seal(base.Add(6 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := wireOf(); got != before+7777 {
		t.Fatalf("the seal did not count the late record: wire %d, want %d", got, before+7777)
	}
}

// An installation from before the billing summaries existed: its
// flow files are summarised by the backfill at the first open, the
// whole of them, and the folder then takes up from the watermark
// without counting anything twice.
func TestBillBackfillCoversFiles(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	rows := foldFixture(base)

	// build the "old" installation with the current code, then take
	// the billing summaries and their mark away again
	old, err := Open(Options{Dir: dir, Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	old.stopFold()
	if err := old.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	if err := old.Seal(base.Add(6 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	want := billDump(t, old)
	if _, err := old.DB.Exec(`DELETE FROM r_bill`); err != nil {
		t.Fatal(err)
	}
	if _, err := old.DB.Exec(`DELETE FROM meta WHERE k = ?`, billKey); err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	s2, err := Open(Options{Dir: dir, Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	s2.stopFold()
	for i := 0; i < 200 && !s2.billReady.Load(); i++ {
		time.Sleep(50 * time.Millisecond)
	}
	if !s2.billReady.Load() {
		t.Fatal("the backfill did not finish")
	}
	if got := billDump(t, s2); got != want {
		t.Fatalf("backfilled summaries differ from the sealed ones:\n%s\n---\n%s", got, want)
	}
	if bs := s2.billStart(); !bs.Equal(base) {
		t.Fatalf("billing summaries reach back to %v, want %v", bs, base)
	}
	// folding on from the watermark counts nothing twice
	if err := s2.foldOnce(base.Add(6 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := billDump(t, s2); got != want {
		t.Fatalf("folding after the backfill counted rows twice:\n%s\n---\n%s", got, want)
	}
}
