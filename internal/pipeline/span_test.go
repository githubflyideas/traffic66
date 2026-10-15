package pipeline

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/store"
)

// A small flow spread over hours (a laptop that slept with a connection
// open): every minute's share stays non-negative and they add up exactly.
// Rounding each minute on its own used to leave the last share below zero,
// which wrapped to nearly 2^64 and broke every sum.
func TestLongSmallFlowShares(t *testing.T) {
	p, st := setup(t)
	now := time.Now().UTC().Truncate(time.Minute)
	start := now.Add(-5*time.Hour - 59*time.Minute)
	p.Ingest([]flow.Record{{Start: start, End: now, Src: netip.MustParseAddr("192.168.1.9"), Dst: netip.MustParseAddr("198.51.100.9"),
		SrcPort: 50000, DstPort: 443, Proto: 6, Bytes: 1467, Packets: 7, L2: true, Mult: 1, SamplingKnown: true,
		Exporter: netip.MustParseAddr("127.0.0.1"), Source: flow.SrcCapture, Direction: 255}})
	var b, w, pk uint64
	for _, r := range p.batch {
		if r.Bytes > 1<<40 || r.Wire > 1<<40 || r.Pkts > 1<<40 {
			t.Fatalf("wrapped share %+v", r)
		}
		b, w, pk = b+r.Bytes, w+r.Wire, pk+r.Pkts
	}
	if w != 1467 || pk != 7 || b != 1467-7*18 {
		t.Fatalf("sums wire %d pkts %d bytes %d", w, pk, b)
	}
	p.FlushRows()
	p.FlushRollups()
	tot, err := st.Totals(store.Query{From: start.Add(-time.Hour), To: now.Add(time.Minute)})
	if err != nil || tot.Wire != 1467 {
		t.Fatalf("totals %+v %v", tot, err)
	}
}

func TestShare(t *testing.T) {
	var done uint64
	for i := 1; i <= 360; i++ {
		done += share(1000, float64(i)/360, done)
	}
	if done != 1000 {
		t.Fatal(done)
	}
	if share(10, 0.5, 9) != 0 {
		t.Fatal("negative share not clamped")
	}
}
