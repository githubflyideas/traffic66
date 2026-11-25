package store

import (
	"testing"
	"time"
)

// A purge takes the conversation summaries with everything else:
// the hourly rows and the daily rows before the time go, the ones
// inside the kept range stay — a purge that left them behind would
// keep answering for records that no longer exist, and billing has
// already shown the way. Deleting everything leaves none at all.
func TestPurgeTakesConversationSummaries(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s, err := Open(Options{Dir: t.TempDir(), Threads: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.stopFold()
	// ten days of conversations, a twenty-day gap, ten more days
	for _, off := range []time.Duration{0, 20 * 24 * time.Hour} {
		if err := s.AppendRows(routeFixture(base.Add(off))); err != nil {
			t.Fatal(err)
		}
	}
	s.wmu.Lock()
	if err := s.rebuildRange(base, base.Add(30*24*time.Hour)); err != nil {
		s.wmu.Unlock()
		t.Fatal(err)
	}
	s.wmu.Unlock()
	// daily summaries for every day the hourly ones cover
	if err := s.convDayRollup(base.Add(30 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	count := func(q string, args ...any) int64 {
		var n int64
		if err := s.DB.QueryRow(q, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	cut := base.Add(10 * 24 * time.Hour)
	kept := base.Add(20 * 24 * time.Hour)
	var convOld, convNew, dayOld, dayNew int64
	convOld = count(`SELECT count(*) FROM r_conv WHERE ts < ?`, cut)
	convNew = count(`SELECT count(*) FROM r_conv WHERE ts >= ?`, kept)
	dayOld = count(`SELECT count(*) FROM r_conv_day WHERE ts < ?`, cut)
	dayNew = count(`SELECT count(*) FROM r_conv_day WHERE ts >= ?`, kept)
	if convOld == 0 || convNew == 0 || dayOld == 0 || dayNew == 0 {
		t.Fatalf("fixture: hourly %d old / %d kept, daily %d old / %d kept, want all four nonzero",
			convOld, convNew, dayOld, dayNew)
	}

	if _, err := s.PurgeBefore(cut); err != nil {
		t.Fatal(err)
	}
	if n := count(`SELECT count(*) FROM r_conv WHERE ts < ?`, cut); n != 0 {
		t.Errorf("%d hourly conversation rows before the purge time left", n)
	}
	if n := count(`SELECT count(*) FROM r_conv_day WHERE ts < ?`, cut); n != 0 {
		t.Errorf("%d daily conversation rows before the purge time left", n)
	}
	if n := count(`SELECT count(*) FROM r_conv WHERE ts >= ?`, kept); n != convNew {
		t.Errorf("kept hourly conversation rows: %d, want %d", n, convNew)
	}
	if n := count(`SELECT count(*) FROM r_conv_day WHERE ts >= ?`, kept); n != dayNew {
		t.Errorf("kept daily conversation rows: %d, want %d", n, dayNew)
	}

	if _, err := s.PurgeBefore(time.Time{}); err != nil {
		t.Fatal(err)
	}
	for _, tb := range []string{"r_conv", "r_conv_day"} {
		if n := count(`SELECT count(*) FROM ` + tb); n != 0 {
			t.Errorf("%s: %d rows after deleting all", tb, n)
		}
	}
}
