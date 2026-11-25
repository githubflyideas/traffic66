package pipeline

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
)

// The counter queue holds 256 batches. With nobody draining it, batches
// past that are dropped — and every dropped snapshot must be counted,
// or a busy minute silently reads as an interface disagreeing with its
// flows.
func TestSubmitCountersCountsDrops(t *testing.T) {
	p, _ := setup(t)
	one := []flow.IfCounters{{
		Time: time.Now(), Exporter: netip.MustParseAddr("192.0.2.1"),
		IfIndex: 1, InOctets: 100, OutOctets: 200,
	}}
	two := []flow.IfCounters{one[0], one[0]}

	for i := 0; i < 256; i++ {
		p.SubmitCounters(one)
	}
	if got := p.DroppedCounters.Load(); got != 0 {
		t.Fatalf("queued batches counted as dropped: %d", got)
	}
	for i := 0; i < 10; i++ {
		p.SubmitCounters(two)
	}
	if got := p.DroppedCounters.Load(); got != 20 {
		t.Fatalf("dropped snapshots = %d, want 20", got)
	}
	p.SubmitCounters(nil)
	if got := p.DroppedCounters.Load(); got != 20 {
		t.Fatalf("empty submit changed the count: %d", got)
	}
}
