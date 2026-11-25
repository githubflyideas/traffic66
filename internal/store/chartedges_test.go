package store

import (
	"math"
	"testing"
	"time"
)

func chartStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func flowRow(ts time.Time, wire uint64) Row {
	return Row{TS: ts, Src: "10.0.0.1", Dst: "10.0.0.2", Client: "10.0.0.1", Server: "10.0.0.2", App: "HTTPS",
		Exporter: "127.0.0.1", InIf: 3, Bytes: wire, Wire: wire, Pkts: 1, Flows: 1}
}

func seriesTotal(se *Series, i int) float64 {
	v := 0.0
	for _, s := range se.Values {
		v += s[i]
	}
	return v
}

// A range that reaches back before the data begins says where the data
// begins, so the page draws a gap there and not traffic of zero.
func TestSeriesMarksWhereTheDataBegins(t *testing.T) {
	s := chartStore(t)
	begin := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Minute)
	var rows []Row
	for m := 0; m < 30; m++ {
		rows = append(rows, flowRow(begin.Add(time.Duration(m)*time.Minute), 7500)) // 1000 bit/s
	}
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	q := Query{From: begin.Add(-30 * time.Minute), To: begin.Add(30 * time.Minute)}
	for _, get := range []func() (*Series, error){
		func() (*Series, error) { return s.SeriesByApp(q, 8) },
		func() (*Series, error) { return s.SeriesBy(q, "if_in", "wire", 8) },
	} {
		se, err := get()
		if err != nil {
			t.Fatal(err)
		}
		if se.DataFrom != begin.UnixMilli() {
			t.Fatalf("data_from %v, want %v", time.UnixMilli(se.DataFrom).UTC(), begin)
		}
	}
	// a range that lies inside the data has nothing to mark
	se, err := s.SeriesByApp(Query{From: begin.Add(10 * time.Minute), To: begin.Add(20 * time.Minute)}, 8)
	if err != nil {
		t.Fatal(err)
	}
	if se.DataFrom != 0 {
		t.Fatalf("data_from %v inside the data", time.UnixMilli(se.DataFrom))
	}
}

// Data that begins in the middle of a bucket is a rate over the part of
// the bucket it covers, not over the whole bucket: no ramp at the start.
func TestFirstBucketCountsOnlyTheCoveredPart(t *testing.T) {
	s := chartStore(t)
	// five-minute buckets (a 6-hour range), data from two minutes into one
	base := time.Now().UTC().Add(-3 * time.Hour).Truncate(5 * time.Minute)
	begin := base.Add(2 * time.Minute)
	var rows []Row
	for m := 0; m < 60; m++ {
		rows = append(rows, flowRow(begin.Add(time.Duration(m)*time.Minute), 7500))
	}
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	q := Query{From: base.Add(-2 * time.Hour), To: base.Add(4 * time.Hour)}
	if s.rollupOK(q, false) {
		t.Skip("range answered from the summaries")
	}
	for name, get := range map[string]func() (*Series, error){
		"app":   func() (*Series, error) { return s.SeriesByApp(q, 8) },
		"if_in": func() (*Series, error) { return s.SeriesBy(q, "if_in", "wire", 8) },
	} {
		se, err := get()
		if err != nil {
			t.Fatal(err)
		}
		if se.Step != 300 {
			t.Fatalf("%s: step %d", name, se.Step)
		}
		i := -1
		for k, ms := range se.Times {
			if ms == base.UnixMilli() {
				i = k
			}
		}
		if i < 0 {
			t.Fatalf("%s: no bucket at %v", name, base)
		}
		first, next := seriesTotal(se, i), seriesTotal(se, i+1)
		if math.Abs(first-1000) > 1 || math.Abs(next-1000) > 1 {
			t.Fatalf("%s: first bucket %.1f bit/s, next %.1f, want 1000 both", name, first, next)
		}
	}
}

// A range that ends in the past in the middle of a minute holds that whole
// minute's row: its last bucket is a rate over the minute, not over the
// seconds before the end. Otherwise yesterday's line jumps up at the end.
func TestPastRangeLastBucketIsNotInflated(t *testing.T) {
	s := chartStore(t)
	begin := time.Now().UTC().Add(-26 * time.Hour).Truncate(time.Minute)
	var rows []Row
	for m := 0; m < 60; m++ {
		rows = append(rows, flowRow(begin.Add(time.Duration(m)*time.Minute), 7500))
	}
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	to := begin.Add(40*time.Minute + 25*time.Second)
	se, err := s.TotalSeries(Query{From: begin.Add(10 * time.Minute), To: to})
	if err != nil {
		t.Fatal(err)
	}
	n := len(se.Times)
	if last := se.Values[0][n-1]; math.Abs(last-1000) > 1 {
		t.Fatalf("last bucket %.1f bit/s, want 1000 (the whole minute)", last)
	}
}

func TestBucketSecs(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	step := 5 * time.Minute
	to := start.Add(time.Hour)
	if got := bucketSecs(start, 0, step, to, time.Time{}); got != 300 {
		t.Errorf("whole bucket %v", got)
	}
	if got := bucketSecs(start, 0, step, to, start.Add(2*time.Minute)); got != 180 {
		t.Errorf("data from 2 minutes in: %v", got)
	}
	// past range ending mid-minute: the minute is whole
	if got := bucketSecs(start, 11, step, start.Add(55*time.Minute+30*time.Second), time.Time{}); got != 60 {
		t.Errorf("past partial bucket %v, want 60", got)
	}
	// a range ending now covers only up to now
	now := time.Now().UTC()
	b := now.Truncate(time.Minute)
	if got := bucketSecs(b, 0, time.Minute, now, time.Time{}); math.Abs(got-max(1, now.Sub(b).Seconds())) > 0.01 {
		t.Errorf("live partial bucket %v", got)
	}
}

// Counters but no flows at all in a direction is an export that is not
// happening, reported as such and not as a deviation of -100%.
func TestReconNoFlowsIsNotADeviation(t *testing.T) {
	s := chartStore(t)
	base := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	for m := 0; m <= 60; m++ {
		ts := base.Add(time.Duration(m) * time.Minute)
		if err := s.AppendCounter(ts, "192.168.108.125", 2198, 1e9, uint64(m)*1e6, uint64(m)*2e6, uint64(m)*1e3, uint64(m)*2e3); err != nil {
			t.Fatal(err)
		}
	}
	// flows only out of the interface, none into it
	var rows []Row
	for m := 0; m < 60; m++ {
		r := flowRow(base.Add(time.Duration(m)*time.Minute), 2e6)
		r.Exporter, r.InIf, r.OutIf = "192.168.108.125", 0, 2198
		rows = append(rows, r)
	}
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	rc, err := s.Recon(Query{From: base, To: base.Add(time.Hour)}, "192.168.108.125", 2198)
	if err != nil {
		t.Fatal(err)
	}
	if !rc.HasCounters {
		t.Fatal("no counters")
	}
	if !rc.InNoFlow || rc.OutNoFlow {
		t.Fatalf("in_no_flow %v out_no_flow %v, want true false", rc.InNoFlow, rc.OutNoFlow)
	}
	// an interface with no flows at all: both directions
	rc, err = s.Recon(Query{From: base, To: base.Add(time.Hour)}, "192.168.108.125", 2207)
	if err != nil {
		t.Fatal(err)
	}
	if rc.HasCounters || rc.InNoFlow || rc.OutNoFlow {
		t.Fatalf("interface without counters marked: %+v", rc.HasCounters)
	}
}
