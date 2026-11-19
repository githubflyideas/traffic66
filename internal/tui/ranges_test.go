package tui

import (
	"testing"
	"time"
)

// The time ranges the t key steps through, and the length each one asks
// the server for, are two lists that must agree.
func TestRangesAndTheirLengths(t *testing.T) {
	if len(ranges) != len(rangeDur) {
		t.Fatalf("%d ranges, %d lengths", len(ranges), len(rangeDur))
	}
	seen := map[string]bool{}
	for i, r := range ranges {
		d, err := time.ParseDuration(r)
		if err != nil || d != rangeDur[i] {
			t.Errorf("%s: length %v, want %v", r, rangeDur[i], d)
		}
		if i > 0 && rangeDur[i] <= rangeDur[i-1] {
			t.Errorf("%s is not longer than %s", r, ranges[i-1])
		}
		seen[r] = true
	}
	for _, r := range []string{"30m", "12h"} {
		if !seen[r] {
			t.Errorf("no %s range", r)
		}
	}
}

// The header shows each range by a short label, the same in every language.
func TestRangeShortLabels(t *testing.T) {
	want := map[string]string{"5m": "5M", "15m": "15M", "30m": "30M", "1h": "1H", "6h": "6H", "12h": "12H", "24h": "1D"}
	for _, r := range ranges {
		if got := rangeShort(r); got != want[r] {
			t.Errorf("%s: %q, want %q", r, got, want[r])
		}
	}
}
