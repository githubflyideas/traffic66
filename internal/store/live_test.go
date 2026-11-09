package store

import (
	"testing"
	"time"
)

func TestSettledTo(t *testing.T) {
	defer SetSettle(Settle())
	now := time.Now()
	// sFlow: data arrives at once, the current minute is read up to now
	SetSettle(2 * time.Second)
	if e := settledTo(now, false); now.Sub(e) > time.Second {
		t.Fatalf("sflow: end %v, want about now", now.Sub(e))
	}
	// NetFlow: only complete minutes, before the delay
	SetSettle(90 * time.Second)
	e := settledTo(now, false)
	if e.Truncate(time.Minute) != e || now.Sub(e) < 90*time.Second || now.Sub(e) > 150*time.Second {
		t.Fatalf("netflow: end %v before now", now.Sub(e))
	}
	// a range in the past is complete throughout
	past := now.Add(-time.Hour)
	if settledTo(past, true) != past {
		t.Fatal("past range moved")
	}
	if m := unsettledMark(settledTo(past, true), past); m != 0 {
		t.Errorf("a past range was marked unsettled at %v", time.UnixMilli(m))
	}
	// a live range is marked, at the boundary and not at its end
	SetSettle(90 * time.Second)
	if m := unsettledMark(settledTo(now, false), now); m == 0 || m >= now.UnixMilli() {
		t.Errorf("mark %v, want a boundary before %v", time.UnixMilli(m), now)
	}
	// a last bucket of a few seconds is left out
	start := now.Truncate(time.Minute).Add(-10 * time.Minute)
	if n := bucketCount(start, start.Add(10*time.Minute+5*time.Second), time.Minute); n != 10 {
		t.Fatalf("buckets %d", n)
	}
}
