package collector

import (
	"net/netip"
	"testing"
	"time"
)

// An sFlow agent's first-seen time is its first datagram, not the zero
// time (shown as 0001-01-01), and later datagrams do not move it.
func TestSFlowSourceFirstSeen(t *testing.T) {
	c := New(nil)
	from := netip.MustParseAddr("192.0.2.7")
	t0 := time.Date(2026, 10, 16, 3, 0, 0, 0, time.UTC)
	c.handleSFlow([]byte{0, 0, 0, 5, 0, 0, 0, 1}, from, t0)
	c.handleSFlow([]byte{0, 0, 0, 5, 0, 0, 0, 1}, from, t0.Add(time.Minute))
	var found bool
	for _, s := range c.Sources() {
		if s.Exporter != from.String() {
			continue
		}
		found = true
		if !s.FirstSeen.Equal(t0) {
			t.Errorf("first seen %v, want %v", s.FirstSeen, t0)
		}
	}
	if !found {
		t.Fatal("the sFlow agent is not listed")
	}
}
