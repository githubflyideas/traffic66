package store

import (
	"math"
	"testing"
	"time"
)

type bucket struct {
	ts      time.Time
	in, out float64
	samples int
}

func rollRates(t *testing.T, snaps func(add func(ts time.Time, ifindex uint32, speed, in, out uint64)), now time.Time) map[int64]bucket {
	t.Helper()
	s, err := Open(Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	snaps(func(ts time.Time, ifindex uint32, speed, in, out uint64) {
		if err := s.AppendCounter(ts, "10.0.0.1", ifindex, speed, in, out, 0, 0); err != nil {
			t.Fatal(err)
		}
	})
	if err := s.RollRates(now); err != nil {
		t.Fatal(err)
	}
	rows, err := s.DB.Query(`SELECT ts, in_bps, out_bps, samples FROM ifr ORDER BY ts`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[int64]bucket{}
	for rows.Next() {
		var b bucket
		if err := rows.Scan(&b.ts, &b.in, &b.out, &b.samples); err != nil {
			t.Fatal(err)
		}
		got[b.ts.Unix()] = b
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return got
}

// A counter climbing at a steady rate must come back as that rate, and the
// bucket it lands in must be the five-minute boundary below its snapshots.
func TestRatesFromSteadyCounter(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 10, 0, 0, time.UTC)
	start := now.Add(-10 * time.Minute) // 12:00
	const perSec = 1_000_000            // bytes
	got := rollRates(t, func(add func(time.Time, uint32, uint64, uint64, uint64)) {
		for i := 0; i < 11; i++ { // 12:00:00 .. 12:05:00, every 30s
			ts := start.Add(time.Duration(i) * 30 * time.Second)
			oct := uint64(i) * 30 * perSec
			add(ts, 7, 10_000_000_000, oct, oct/2)
		}
	}, now)
	if len(got) != 2 {
		t.Fatalf("want the 12:00 and 12:05 buckets, got %d: %v", len(got), got)
	}
	first, ok := got[start.Unix()]
	if !ok {
		t.Fatalf("no bucket at %s: %v", start, got)
	}
	if want := float64(perSec * 8); math.Abs(first.in-want) > 1 {
		t.Errorf("in_bps = %v, want %v", first.in, want)
	}
	if want := float64(perSec * 8 / 2); math.Abs(first.out-want) > 1 {
		t.Errorf("out_bps = %v, want %v", first.out, want)
	}
	// Ten of the eleven snapshots open an interval; nine of those land in
	// the first bucket, the one ending at 12:05:00 lands in the second.
	if first.samples != 9 {
		t.Errorf("samples = %d, want 9", first.samples)
	}
}

// A counter that goes backwards is a wrap or a reboot and cannot be told
// apart from here, so the interval it opens is dropped rather than guessed
// at. The bucket survives on its remaining intervals.
func TestRatesDropCounterReset(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 5, 0, 0, time.UTC)
	start := now.Add(-5 * time.Minute)
	const perSec = 1_000_000
	got := rollRates(t, func(add func(time.Time, uint32, uint64, uint64, uint64)) {
		for i := 0; i < 10; i++ {
			ts := start.Add(time.Duration(i) * 30 * time.Second)
			oct := uint64(i) * 30 * perSec
			if i >= 6 {
				oct -= 6 * 30 * perSec // the device restarted at 12:03
			}
			add(ts, 7, 10_000_000_000, oct, oct)
		}
	}, now)
	b, ok := got[start.Unix()]
	if !ok {
		t.Fatalf("the bucket was thrown away whole: %v", got)
	}
	// Nine intervals, and the one crossing the reset is not one of them.
	if b.samples != 8 {
		t.Errorf("samples = %d, want 8 (nine intervals less the reset)", b.samples)
	}
	if want := float64(perSec * 8); math.Abs(b.in-want) > 1 {
		t.Errorf("in_bps = %v, want %v: the reset moved the rate", b.in, want)
	}
}

// A rate above the interface speed is a counter reset the backwards test
// cannot see. It is dropped for the same reason.
func TestRatesDropAboveSpeed(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 5, 0, 0, time.UTC)
	start := now.Add(-5 * time.Minute)
	got := rollRates(t, func(add func(time.Time, uint32, uint64, uint64, uint64)) {
		add(start, 7, 1_000_000_000, 0, 0)
		// 1 Gb/s for thirty seconds is 3.75 GB, which is the line exactly.
		add(start.Add(30*time.Second), 7, 1_000_000_000, 30*125_000_000, 0)
		add(start.Add(60*time.Second), 7, 1_000_000_000, 1<<42, 0) // far over: dropped
	}, now)
	b, ok := got[start.Unix()]
	if !ok {
		t.Fatalf("no bucket: %v", got)
	}
	if b.samples != 1 {
		t.Errorf("samples = %d, want 1", b.samples)
	}
	if want := 1e9; math.Abs(b.in-want) > 1 {
		t.Errorf("in_bps = %v, want %v", b.in, want)
	}
}

// Nothing invents a zero: an interface with a gap in collection has no
// bucket there, rather than a bucket saying it carried no traffic.
func TestRatesLeaveGapsEmpty(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC)
	start := now.Add(-30 * time.Minute)
	got := rollRates(t, func(add func(time.Time, uint32, uint64, uint64, uint64)) {
		add(start, 7, 0, 0, 0)
		add(start.Add(30*time.Second), 7, 0, 30_000_000, 0)
		// Twenty minutes later the collector is back. The interval across
		// the hole is longer than rateMaxGap and is not a measurement.
		add(start.Add(25*time.Minute), 7, 0, 60_000_000, 0)
		add(start.Add(25*time.Minute+30*time.Second), 7, 0, 90_000_000, 0)
	}, now)
	if len(got) != 2 {
		t.Fatalf("want two buckets with eighteen empty between them, got %d: %v", len(got), got)
	}
	if _, ok := got[start.Add(10*time.Minute).Unix()]; ok {
		t.Error("a bucket inside the gap was filled in")
	}
	// The bucket the collector came back in rests on its own interval
	// alone. Folding the twenty-minute hole into it would spread half an
	// hour of counting over one five-minute bucket.
	back, ok := got[start.Add(25*time.Minute).Unix()]
	if !ok {
		t.Fatalf("no bucket where collection resumed: %v", got)
	}
	if back.samples != 1 {
		t.Errorf("samples = %d, want 1: the gap was counted as a measurement", back.samples)
	}
	if want := 8e6; math.Abs(back.in-want) > 1 {
		t.Errorf("in_bps = %v, want %v", back.in, want)
	}
}

// Running the pass again over a span it has already written replaces its
// own work instead of doubling it.
func TestRatesRollTwice(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 5, 0, 0, time.UTC)
	start := now.Add(-5 * time.Minute)
	s, err := Open(Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for i := 0; i < 10; i++ {
		oct := uint64(i) * 30 * 1_000_000
		if err := s.AppendCounter(start.Add(time.Duration(i)*30*time.Second), "10.0.0.1", 7, 0, oct, oct, 0, 0); err != nil {
			t.Fatal(err)
		}
	}
	var first, second float64
	for _, p := range []*float64{&first, &second} {
		if err := s.RollRates(now); err != nil {
			t.Fatal(err)
		}
		var n int
		if err := s.DB.QueryRow(`SELECT count(*), coalesce(sum(in_bps), 0) FROM ifr`).Scan(&n, p); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("%d rows after a pass, want 1", n)
		}
	}
	if first != second {
		t.Errorf("the second pass changed the rate: %v then %v", first, second)
	}
}

func putRate(t *testing.T, s *Store, ts time.Time, ifindex uint32, in, out float64, samples int) {
	t.Helper()
	if _, err := s.DB.Exec(`INSERT INTO ifr VALUES (?, '10.0.0.1', ?, ?, ?, ?)`, ts.UTC(), ifindex, in, out, samples); err != nil {
		t.Fatal(err)
	}
}

// The billed figure is the sample left standing after the top five per cent
// are discarded — position ceil(0.95n) counting up. A full month of
// five-minute buckets is 8640, and the 8208th is what the bill says. This
// test exists because the whole product claim rests on that being the
// number a carrier would arrive at independently.
func TestBillIsTheSampleAfterDiscardingTopFivePercent(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	const n = 8640 // thirty days of five-minute buckets
	for i := 0; i < n; i++ {
		// 1 Mb/s per step, so the sample's rank is readable in the answer
		putRate(t, s, from.Add(time.Duration(i)*5*time.Minute), 1, float64(i+1)*1e6, 0, 10)
	}
	bills, err := s.Bills(from, from.Add(n*5*time.Minute), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(bills) != 1 {
		t.Fatalf("%d interfaces, want 1", len(bills))
	}
	b := bills[0]
	if want := 8208 * 1e6; b.In95 != want {
		t.Errorf("in95 = %v, want %v (the 8208th of 8640 samples)", b.In95, want)
	}
	if b.Billed != b.In95 || b.Dir != "in" {
		t.Errorf("billed %v on %q, want the inbound percentile", b.Billed, b.Dir)
	}
	if b.Buckets != n || b.Possible != n || b.Coverage() != 1 {
		t.Errorf("coverage %v from %d of %d buckets, want the whole period", b.Coverage(), b.Buckets, b.Possible)
	}
	if b.Samples != n*10 {
		t.Errorf("samples = %d, want %d", b.Samples, n*10)
	}
}

// In and out are percentiled separately and the greater is charged, which
// is how a carrier words it. Charging the percentile of the larger of each
// pair would be a different and higher number.
func TestBillChargesTheGreaterDirection(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	// Outbound is quiet except for one hour, when it is far above inbound.
	// Its 95th percentile stays below the steady inbound one all the same.
	for i := 0; i < 288; i++ {
		out := 1e6
		if i < 12 {
			out = 900e6
		}
		putRate(t, s, from.Add(time.Duration(i)*5*time.Minute), 1, 100e6, out, 10)
	}
	bills, err := s.Bills(from, from.Add(24*time.Hour), "")
	if err != nil {
		t.Fatal(err)
	}
	b := bills[0]
	if b.Dir != "in" || b.Billed != 100e6 {
		t.Errorf("billed %v on %q, want 100000000 on in: the outbound spike is inside the discarded five per cent", b.Billed, b.Dir)
	}
	if b.OutMax != 900e6 {
		t.Errorf("out_max = %v, want the spike to still be visible as a peak", b.OutMax)
	}
}

// A period only half measured still gives a number, and the number is
// taken over what was measured. Coverage is what stops that being quietly
// misread as a month.
func TestBillCoverageShowsWhatIsMissing(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 144; i++ { // the first half of one day
		putRate(t, s, from.Add(time.Duration(i)*5*time.Minute), 1, 500e6, 0, 10)
	}
	bills, err := s.Bills(from, from.Add(24*time.Hour), "")
	if err != nil {
		t.Fatal(err)
	}
	b := bills[0]
	if b.Billed != 500e6 {
		t.Errorf("billed = %v, want 500000000: the percentile is over what was measured", b.Billed)
	}
	if b.Buckets != 144 || b.Possible != 288 || b.Coverage() != 0.5 {
		t.Errorf("coverage %v from %d of %d, want half the day", b.Coverage(), b.Buckets, b.Possible)
	}
}

// The period is given as an interval, so a calendar month, a thirty-day
// span and a cycle starting on the third are all the same call. Nothing in
// here knows about months, which is why no time zone is baked in: the
// caller hands over the boundaries it means.
func TestBillTakesAnyInterval(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 288*10; i++ {
		rate := 10e6
		if i >= 288*2 && i < 288*3 { // the third of the month is busy
			rate = 800e6
		}
		putRate(t, s, base.Add(time.Duration(i)*5*time.Minute), 1, rate, 0, 10)
	}
	cycle, err := s.Bills(base.AddDate(0, 0, 2), base.AddDate(0, 0, 3), "")
	if err != nil {
		t.Fatal(err)
	}
	if cycle[0].Billed != 800e6 {
		t.Errorf("the third alone billed %v, want 800000000", cycle[0].Billed)
	}
	whole, err := s.Bills(base, base.AddDate(0, 0, 10), "")
	if err != nil {
		t.Fatal(err)
	}
	// One busy day in ten is 10% of the samples, so the percentile does not
	// reach it: the top 5% are discarded, the rest of that day is not.
	if whole[0].Billed != 800e6 {
		t.Errorf("the ten days billed %v, want 800000000", whole[0].Billed)
	}
	if whole[0].Possible != 2880 {
		t.Errorf("possible = %d, want 2880", whole[0].Possible)
	}
}

func TestBillRejectsBackwardsPeriod(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Now()
	if _, err := s.Bills(now, now.Add(-time.Hour), ""); err == nil {
		t.Error("Bills accepted a period that ends before it starts")
	}
}
