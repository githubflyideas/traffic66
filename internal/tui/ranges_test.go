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
