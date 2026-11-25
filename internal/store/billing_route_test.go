package store

import (
	"net/netip"
	"testing"
	"time"
)

// billFixture is days of flows around two billed sets: a product
// line of two addresses with traffic between them, a single VIP,
// and a v6 line — plus traffic that touches no set, and a duplicate
// copy that only counts when duplicates are asked for. One row per
// flow per hour, thirteen minutes into the hour, so the hourly
// summaries have minutes to fold.
func billFixture(base time.Time, days int) []Row {
	const (
		u1, u2     = "198.51.100.7", "198.51.100.8"
		a, b, c    = "203.0.113.10", "203.0.113.11", "203.0.113.12"
		u6, a6, b6 = "2001:db8:2::7", "2001:db8:1::10", "2001:db8:1::11"
	)
	var rows []Row
	for h := 0; h < days*24; h++ {
		ts := base.Add(time.Duration(h)*time.Hour + 13*time.Minute)
		w := uint64(1000 + h)
		mk := func(src, dst string, wire uint64, dup bool) Row {
			return Row{TS: ts, Src: src, Dst: dst, Client: src, Server: dst,
				Bytes: wire * 3 / 4, Wire: wire, Pkts: 10, Flows: 1,
				Exporter: "10.0.0.1", Dir: DirTransit, Dup: dup}
		}
		rows = append(rows,
			mk(u1, a, w, false),       // in to A
			mk(a, u1, w+100, false),   // out of A
			mk(a, b, w+200, false),    // inside the line, A -> B
			mk(b, a, w+300, false),    // inside the line, B -> A
			mk(u2, b, w+400, false),   // in to B
			mk(b, u2, w+500, false),   // out of B
			mk(u1, c, w+600, false),   // the single VIP, in
			mk(c, u2, w+700, false),   // the single VIP, out
			mk(u1, u2, w+800, false),  // touches no billed set
			mk(u6, a6, w+900, false),  // the v6 line, in
			mk(a6, b6, w+1000, false), // inside the v6 line
			mk(u1, a, w+2000, true),   // a duplicate copy of a flow
		)
	}
	return rows
}

var (
	billLine   = []netip.Addr{netip.MustParseAddr("203.0.113.10"), netip.MustParseAddr("203.0.113.11")}
	billSingle = []netip.Addr{netip.MustParseAddr("203.0.113.10")}
	billLine6  = []netip.Addr{netip.MustParseAddr("2001:db8:1::10"), netip.MustParseAddr("2001:db8:1::11")}
)

// billFigureEq compares two billing answers: the settled figures and
// the member split always, the series too when series is set (the
// ranges it is set for are bucketed by the hour or wider on both
// paths, so the buckets themselves must agree, not just their sums).
func billFigureEq(t *testing.T, name string, want, got *Billing, series bool) {
	t.Helper()
	if want.InBytes != got.InBytes || want.OutBytes != got.OutBytes || want.Bytes != got.Bytes {
		t.Errorf("%s: settled in/out/bytes = %d/%d/%d, want %d/%d/%d",
			name, got.InBytes, got.OutBytes, got.Bytes, want.InBytes, want.OutBytes, want.Bytes)
	}
	if len(want.Members) != len(got.Members) {
		t.Fatalf("%s: %d members, want %d", name, len(got.Members), len(want.Members))
	}
	for i := range want.Members {
		if want.Members[i] != got.Members[i] {
			t.Errorf("%s: member %s = %+v, want %+v", name, want.Members[i].Addr, got.Members[i], want.Members[i])
		}
	}
	if want.SettledTo != got.SettledTo || want.DataFrom != got.DataFrom {
		t.Errorf("%s: marks settled/data_from = %d/%d, want %d/%d", name, got.SettledTo, got.DataFrom, want.SettledTo, want.DataFrom)
	}
	if !series {
		return
	}
	if want.Step != got.Step {
		t.Fatalf("%s: step %d, want %d", name, got.Step, want.Step)
	}
	if len(want.Times) != len(got.Times) {
		t.Fatalf("%s: %d buckets, want %d", name, len(got.Times), len(want.Times))
	}
	for i := range want.Times {
		if want.Times[i] != got.Times[i] || want.In[i] != got.In[i] || want.Out[i] != got.Out[i] || want.Total[i] != got.Total[i] {
			t.Errorf("%s: bucket %d = %dms in/out/total %.1f/%.1f/%.1f, want %dms %.1f/%.1f/%.1f",
				name, i, got.Times[i], got.In[i], got.Out[i], got.Total[i],
				want.Times[i], want.In[i], want.Out[i], want.Total[i])
		}
	}
}

// A range the flow records no longer cover is billed from the
// billing summaries whole, and the figures are the records'
// figures: the settled totals, the member split and the inside
// traffic counted once, for a product line, a single VIP and a v6
// line, with duplicates in and out — the records' answers taken
// first with the summaries held out of it, then retention moves
// the records' start past the ranges.
func TestBillingSummaryMatchesRecords(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s, err := Open(Options{Dir: t.TempDir(), Threads: 2, RawDays: 7, RollupDays: 180})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.stopFold()
	if err := s.AppendRows(billFixture(base, 10)); err != nil {
		t.Fatal(err)
	}
	for d := 1; d <= 9; d++ {
		if err := s.Seal(base.Add(time.Duration(d) * 24 * time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.foldOnce(base.Add(10 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}

	type billCase struct {
		name   string
		q      Query
		set    []netip.Addr
		series bool
	}
	cases := []billCase{
		// two days, bucketed by the quarter hour on the records
		// path: the figures must agree, the buckets cannot
		{"line-2d", Query{From: base.Add(24 * time.Hour), To: base.Add(72 * time.Hour)}, billLine, false},
		{"single-2d", Query{From: base.Add(24 * time.Hour), To: base.Add(72 * time.Hour)}, billSingle, false},
		// ten days, hourly buckets on both paths
		{"line-10d", Query{From: base, To: base.Add(10 * 24 * time.Hour)}, billLine, true},
		{"single-10d", Query{From: base, To: base.Add(10 * 24 * time.Hour)}, billSingle, true},
		{"line6-10d", Query{From: base, To: base.Add(10 * 24 * time.Hour)}, billLine6, true},
		{"line-10d-dup", Query{From: base, To: base.Add(10 * 24 * time.Hour), IncludeDup: true}, billLine, true},
		// a viewer nine hours ahead: whole-hour offset, hourly
		// buckets still align with the summaries' hours
		{"line-jst", Query{From: base.Add(24 * time.Hour), To: base.Add(9 * 24 * time.Hour), ZoneOffset: 9 * time.Hour}, billLine, true},
		// a range the records still cover after retention
		{"line-recent", Query{From: base.Add(9 * 24 * time.Hour), To: base.Add(10 * 24 * time.Hour)}, billLine, true},
	}
	// the records' answers, with the summaries held out of it
	s.billReady.Store(false)
	want := make([]*Billing, len(cases))
	for i, c := range cases {
		b, err := s.Billing(c.q, c.set)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		want[i] = b
	}
	s.billReady.Store(true)

	if err := s.Retain(base.Add(10 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	// the records' start is the first row of the first file that
	// straddles the cut — seals end on the hour, so it sits an hour
	// before the seven days, not on them
	if ds := s.DetailStart(); !ds.Equal(base.Add(71*time.Hour + 13*time.Minute)) {
		t.Fatalf("records start %v, want %v", ds, base.Add(71*time.Hour+13*time.Minute))
	}
	for i, c := range cases {
		if c.name == "line-recent" {
			if s.billingSummary(c.q) {
				t.Fatalf("%s: answered from the summaries, want the records", c.name)
			}
		} else if !s.billingSummary(c.q) {
			t.Fatalf("%s: answered from the records, want the summaries", c.name)
		}
		got, err := s.Billing(c.q, c.set)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		billFigureEq(t, c.name, want[i], got, c.series)
	}
	// the short range is bucketed by the hour on the summaries path
	got, err := s.Billing(cases[0].q, cases[0].set)
	if err != nil {
		t.Fatal(err)
	}
	if got.Step != 3600 {
		t.Errorf("summaries path step = %d, want 3600 (hourly at finest)", got.Step)
	}
}

// The upgrade an installed base actually makes: a store holding
// thirty days of records, opened by the new version, backfills its
// billing summaries from the whole of the records — a retention
// pass that arrives while the backfill is still reading does not
// take the files from under it — and only then does retention cut
// the records to seven days. The month's bill afterwards is the
// month's bill before, to the byte.
func TestBillingUpgradeKeepsMonth(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	month := Query{From: base, To: base.Add(30 * 24 * time.Hour)}

	// the old installation: thirty days of records, and no billing
	// summaries — theirs and their mark are taken away again below
	old, err := Open(Options{Dir: dir, Threads: 2, RawDays: 30, RollupDays: 180})
	if err != nil {
		t.Fatal(err)
	}
	old.stopFold()
	if err := old.AppendRows(billFixture(base, 30)); err != nil {
		t.Fatal(err)
	}
	for d := 1; d <= 29; d++ {
		if err := old.Seal(base.Add(time.Duration(d) * 24 * time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if err := old.foldOnce(base.Add(30 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	old.billReady.Store(false) // the records' figures, as the old version gives them
	wantLine, err := old.Billing(month, billLine)
	if err != nil {
		t.Fatal(err)
	}
	wantSingle, err := old.Billing(month, billSingle)
	if err != nil {
		t.Fatal(err)
	}
	old.billReady.Store(true)
	if _, err := old.DB.Exec(`DELETE FROM r_bill`); err != nil {
		t.Fatal(err)
	}
	if _, err := old.DB.Exec(`DELETE FROM meta WHERE k = ?`, billKey); err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	// the new version opens the store: the backfill starts, and a
	// retention pass fires while it is still reading
	s, err := Open(Options{Dir: dir, Threads: 2, RawDays: 7, RollupDays: 180})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.stopFold()
	if err := s.Retain(base.Add(30 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 600 && !s.billReady.Load(); i++ {
		time.Sleep(50 * time.Millisecond)
	}
	if !s.billReady.Load() {
		t.Fatal("the backfill did not finish")
	}
	if bs := s.billStart(); !bs.Equal(base) {
		t.Fatalf("billing summaries reach back to %v, want %v (the whole of the records)", bs, base)
	}
	// now retention may cut, and does: seven days of records remain
	if err := s.Retain(base.Add(30 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if ds := s.DetailStart(); ds.Before(base.Add(22 * 24 * time.Hour)) {
		t.Fatalf("records start %v, want it cut to the last seven days", ds)
	}
	if !s.billingSummary(month) {
		t.Fatal("the month is not answered from the summaries")
	}
	gotLine, err := s.Billing(month, billLine)
	if err != nil {
		t.Fatal(err)
	}
	billFigureEq(t, "line-month", wantLine, gotLine, true)
	gotSingle, err := s.Billing(month, billSingle)
	if err != nil {
		t.Fatal(err)
	}
	billFigureEq(t, "single-month", wantSingle, gotSingle, true)
}
