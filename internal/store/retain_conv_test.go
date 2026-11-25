package store

import (
	"testing"
	"time"
)

// The hourly conversation summaries are kept thirty days. Before a
// day's go, its daily summary — the day's largest conversations — is
// made from them, and a Top that starts before the hourly summaries
// reach back is answered from the daily ones, with the records' own
// answer for that day. The daily summaries are kept with the other
// summaries, 180 days by default, and the defaults are seven days of
// records and 180 of summaries.
func TestConvRetentionAndDailyRoute(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s, err := Open(Options{Dir: t.TempDir(), Threads: 2, RawDays: 7, RollupDays: 180})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.stopFold()
	for d := 0; d < 40; d += 10 {
		if err := s.AppendRows(routeFixture(base.Add(time.Duration(d) * 24 * time.Hour))); err != nil {
			t.Fatal(err)
		}
	}
	s.wmu.Lock()
	if err := s.rebuildRange(base, base.Add(40*24*time.Hour)); err != nil {
		s.wmu.Unlock()
		t.Fatal(err)
	}
	s.wmu.Unlock()
	for d := 1; d <= 39; d++ {
		if err := s.Seal(base.Add(time.Duration(d) * 24 * time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.foldOnce(base.Add(40 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}

	// day 1, whole: the records' answers, taken before retention
	day1 := Query{From: base.Add(24 * time.Hour), To: base.Add(48 * time.Hour)}
	neg := []Filter{{Field: "country", Value: "ZZ", Neg: true}}
	day1Neg := Query{From: day1.From, To: day1.To, Filters: neg}
	want := map[string]string{}
	for _, dim := range []string{"conv", "service", "threat"} {
		rows, err := s.TopNBy(day1Neg, dim, 20, "wire")
		if err != nil {
			t.Fatal(err)
		}
		want[dim] = topDump(rows, false)
	}

	if err := s.Retain(base.Add(40 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	// the hourly summaries now reach back thirty days, no further
	if cs := s.convStart(); !cs.Equal(base.Add(10 * 24 * time.Hour)) {
		t.Fatalf("hourly summaries reach back to %v, want %v", cs, base.Add(10*24*time.Hour))
	}
	// the daily summaries cover the ten days that rolled up
	var days int64
	if err := s.DB.QueryRow(`SELECT count(DISTINCT ts) FROM r_conv_day`).Scan(&days); err != nil {
		t.Fatal(err)
	}
	if days != 10 {
		t.Fatalf("daily summaries cover %d days, want 10", days)
	}
	// a second pass makes nothing again and deletes nothing more
	var dayRows int64
	s.DB.QueryRow(`SELECT count(*) FROM r_conv_day`).Scan(&dayRows)
	if err := s.Retain(base.Add(40 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	var dayRows2 int64
	s.DB.QueryRow(`SELECT count(*) FROM r_conv_day`).Scan(&dayRows2)
	if dayRows2 != dayRows {
		t.Fatalf("second retention changed the daily summaries: %d -> %d", dayRows, dayRows2)
	}
	// day 1 is answered from the daily summaries, and the answer is
	// the records' answer for that day
	for _, dim := range []string{"conv", "service", "threat"} {
		src, sf, st := s.TopSource(day1, dim)
		if src != "rollup_day" {
			t.Fatalf("%s: source %q, want rollup_day", dim, src)
		}
		if !sf.Equal(day1.From) || !st.Equal(day1.To) {
			t.Fatalf("%s: covered %v..%v, want the day whole", dim, sf, st)
		}
		rows, err := s.TopNBy(day1, dim, 20, "wire")
		if err != nil {
			t.Fatal(err)
		}
		if got := topDump(rows, false); got != want[dim] {
			t.Errorf("%s:\ndaily summaries:\n%s\nrecords:\n%s", dim, got, want[dim])
		}
	}
	// at 230 days the daily summaries are past their 180 too
	if err := s.Retain(base.Add(230 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	var left int64
	s.DB.QueryRow(`SELECT count(*) FROM r_conv_day`).Scan(&left)
	if left != 0 {
		t.Fatalf("daily summaries past 180 days: %d rows left", left)
	}
}

// The defaults an installation runs with when nothing is set: seven
// days of flow records, 180 of summaries.
func TestRetentionDefaults(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.opt.RawDays != 7 || s.opt.RollupDays != 180 {
		t.Fatalf("defaults are %d days of records, %d of summaries; want 7 and 180", s.opt.RawDays, s.opt.RollupDays)
	}
}
