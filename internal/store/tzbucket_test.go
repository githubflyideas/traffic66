package store

import (
	"testing"
	"time"
)

// A day bucket must start at midnight on the viewer's clock. Cut on UTC, a
// Japanese day starts at nine in the morning, so every day is split across
// two columns and neither holds a day's traffic.
func TestDayBucketsStartAtLocalMidnight(t *testing.T) {
	tokyo := 9 * time.Hour
	// 2026-03-04 02:00 UTC is 11:00 on the 4th in Tokyo
	ts := time.Date(2026, 3, 4, 2, 0, 0, 0, time.UTC)
	q := Query{ZoneOffset: tokyo}
	got := q.localTrunc(ts, 24*time.Hour)
	// midnight on the 4th in Tokyo is 15:00 on the 3rd, UTC
	want := time.Date(2026, 3, 3, 15, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("day bucket starts %v, want %v (local midnight)", got, want)
	}
	// and on UTC it is unchanged
	if got := (Query{}).localTrunc(ts, 24*time.Hour); !got.Equal(time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("a viewer on UTC got %v", got)
	}
}

// Zones that are not whole hours exist and are not rare.
func TestHalfHourZones(t *testing.T) {
	for _, c := range []struct {
		name string
		off  time.Duration
		want time.Time
	}{
		// 20:00 UTC on the 4th is already past midnight on the 5th in all
		// three, so the day they fall in is the 5th, local.
		{"India +05:30", 5*time.Hour + 30*time.Minute, time.Date(2026, 3, 4, 18, 30, 0, 0, time.UTC)},
		{"Nepal +05:45", 5*time.Hour + 45*time.Minute, time.Date(2026, 3, 4, 18, 15, 0, 0, time.UTC)},
		{"Chatham +12:45", 12*time.Hour + 45*time.Minute, time.Date(2026, 3, 4, 11, 15, 0, 0, time.UTC)},
	} {
		q := Query{ZoneOffset: c.off}
		got := q.localTrunc(time.Date(2026, 3, 4, 20, 0, 0, 0, time.UTC), 24*time.Hour)
		if !got.Equal(c.want) {
			t.Errorf("%s: day starts %v, want %v", c.name, got, c.want)
		}
	}
}

// Zones behind UTC go the other way.
func TestZoneBehindUTC(t *testing.T) {
	q := Query{ZoneOffset: -5 * time.Hour} // New York in winter
	// 2026-03-04 02:00 UTC is still the 3rd, 21:00, in New York
	got := q.localTrunc(time.Date(2026, 3, 4, 2, 0, 0, 0, time.UTC), 24*time.Hour)
	want := time.Date(2026, 3, 3, 5, 0, 0, 0, time.UTC) // midnight on the 3rd, New York
	if !got.Equal(want) {
		t.Errorf("day starts %v, want %v", got, want)
	}
}

// The SQL has to cut rows on the same clock as the columns they go into,
// or rows land in a column that starts somewhere else.
func TestBucketSQLMatchesTruncation(t *testing.T) {
	if got := (Query{}).bucketSQL(24 * time.Hour); got != "time_bucket(INTERVAL '86400 seconds', ts)" {
		t.Errorf("a viewer on UTC gets %q, want the plain expression", got)
	}
	got := Query{ZoneOffset: 9 * time.Hour}.bucketSQL(24 * time.Hour)
	want := "time_bucket(INTERVAL '86400 seconds', ts + INTERVAL '32400 seconds') - INTERVAL '32400 seconds'"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// The six-hour step was called out as well: it splits a day into four, and
// on UTC those quarters start at nine, three, nine and three local.
func TestSixHourBucketsAreLocal(t *testing.T) {
	q := Query{ZoneOffset: 9 * time.Hour}
	got := q.localTrunc(time.Date(2026, 3, 4, 2, 0, 0, 0, time.UTC), 6*time.Hour)
	// 11:00 Tokyo falls in the bucket starting 06:00 Tokyo = 21:00 UTC on the 3rd
	want := time.Date(2026, 3, 3, 21, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("six-hour bucket starts %v, want %v", got, want)
	}
}

// End to end: a day's traffic on the Tokyo clock must arrive as one
// column, not as two partial ones. This is the case the report came from.
func TestTokyoDayIsOneColumn(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	tokyo := 9 * time.Hour
	// midnight on 2026-03-04 in Tokyo, as UTC
	dayStart := time.Date(2026, 3, 3, 15, 0, 0, 0, time.UTC)
	var rows []Row
	// one row an hour through that Tokyo day
	for h := 0; h < 24; h++ {
		rows = append(rows, Row{TS: dayStart.Add(time.Duration(h) * time.Hour),
			Src: "10.0.0.1", Dst: "10.0.0.2", Client: "10.0.0.1", Server: "10.0.0.2",
			App: "https", Exporter: "127.0.0.1", Bytes: 1000, Wire: 1000, Pkts: 1, Flows: 1})
	}
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}

	// A range long enough that the step is a day (over 75 days: below that
	// StepFor picks six hours). The filter keeps the query on the raw rows
	// rather than the rollups, which only the pipeline writes.
	q := Query{From: dayStart.Add(-50 * 24 * time.Hour), To: dayStart.Add(50 * 24 * time.Hour),
		ZoneOffset: tokyo, Filters: []Filter{{Field: "app", Value: "https"}}}
	se, err := s.SeriesByApp(q, 10)
	if err != nil {
		t.Fatal(err)
	}
	if se.Step != 86400 {
		t.Fatalf("step is %d seconds, want a day", se.Step)
	}
	// count the columns that got any traffic
	var hit []int
	for i := range se.Times {
		var v float64
		for _, series := range se.Values {
			v += series[i]
		}
		if v > 0 {
			hit = append(hit, i)
		}
	}
	if len(hit) != 1 {
		t.Errorf("the day's traffic landed in %d columns, want 1", len(hit))
	}
	if len(hit) > 0 {
		if got := time.UnixMilli(se.Times[hit[0]]).UTC(); !got.Equal(dayStart) {
			t.Errorf("the column starts %v, want %v (midnight in Tokyo)", got, dayStart)
		}
	}
}
