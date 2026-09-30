package dnsres

import (
	"testing"
	"time"
)

func TestBudget(t *testing.T) {
	r := New(Options{PerSecond: 2, Upstream: "127.0.0.1:1"})
	ips := []string{"192.0.2.1", "192.0.2.2", "192.0.2.3", "192.0.2.4"}
	r.Lookup(ips, 50*time.Millisecond)
	if r.Queries != 2 || r.Skipped != 2 {
		t.Fatalf("queries %d skipped %d", r.Queries, r.Skipped)
	}
}
