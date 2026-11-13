package pipeline

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/store"
)

// A device sends a long flow that started before anything stored: its
// rate is drawn right away, not multiplied by up to 60 for a minute.
func TestBackfillNoSpike(t *testing.T) {
	p, st := setup(t)
	now := time.Now().UTC().Truncate(time.Minute)
	exp := netip.MustParseAddr("10.0.0.1")
	rec := func(start, end time.Time, bytes uint64) flow.Record {
		return flow.Record{Start: start, End: end, Src: netip.MustParseAddr("10.1.1.1"), Dst: netip.MustParseAddr("198.51.100.9"),
			SrcPort: 50000, DstPort: 443, Proto: 6, Bytes: bytes, Packets: bytes/1000 + 1, Mult: 1, SamplingKnown: true, Exporter: exp, Source: flow.SrcNetFlow9}
	}
	p.Ingest([]flow.Record{rec(now.Add(-10*time.Minute), now.Add(-10*time.Minute+30*time.Second), 1000)})
	p.FlushRows()
	p.FlushRollups()
	q := store.Query{From: now.Add(-6 * time.Hour), To: now}
	if _, err := st.SeriesByApp(q, 5); err != nil { // reads (and keeps) the start of the data
		t.Fatal(err)
	}
	// 60 minutes at 1 Mbit/s, ending 20 minutes ago
	p.Ingest([]flow.Record{rec(now.Add(-80*time.Minute), now.Add(-20*time.Minute), 60*60*1e6/8)})
	p.FlushRows()
	p.FlushRollups()
	se, err := st.SeriesByApp(q, 5)
	if err != nil {
		t.Fatal(err)
	}
	var peak float64
	for _, v := range se.Values {
		for _, x := range v {
			peak = max(peak, x)
		}
	}
	if peak > 1.05e6 || peak < 0.95e6 {
		t.Fatalf("peak %.0f bit/s, want about 1000000", peak)
	}
}
