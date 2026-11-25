package pipeline

import (
	"fmt"
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/store"
)

// benchPipeline is setup without a *testing.T, for benchmarks.
func benchPipeline(b *testing.B) *Pipeline {
	st, err := store.Open(store.Options{Dir: b.TempDir(), RawDays: 30})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { st.Close() })
	inv := enrich.NewInventory()
	inv.Parse("net 10.0.0.0/8 LAN\nnet 203.0.113.0/28 DMZ\n")
	return New(Config{}, st, inv, enrich.NewASNDB(), enrich.NewThreats())
}

// benchRecords builds flows that look like a busy minute: many clients and
// ports behind a handful of exporters, each flow inside one minute.
func benchRecords(n int) []flow.Record {
	ts := time.Now().UTC().Truncate(time.Minute)
	exporters := []netip.Addr{
		netip.MustParseAddr("10.0.0.1"),
		netip.MustParseAddr("10.0.0.2"),
		netip.MustParseAddr("10.0.0.3"),
		netip.MustParseAddr("10.0.0.4"),
	}
	recs := make([]flow.Record, 0, n)
	for i := 0; i < n; i++ {
		cli := netip.AddrFrom4([4]byte{10, 1, byte(i >> 8), byte(i)})
		srv := netip.AddrFrom4([4]byte{198, 51, 100, byte(i % 251)})
		recs = append(recs, flow.Record{
			Start: ts, End: ts.Add(30 * time.Second),
			Src: cli, Dst: srv, SrcPort: uint16(30000 + i%20000), DstPort: 443,
			Proto: 6, Bytes: 100 * 1482, Packets: 100, Mult: 10, SamplingKnown: true,
			Exporter: exporters[i%len(exporters)], Source: flow.SrcNetFlow9, Direction: 0,
		})
	}
	return recs
}

// BenchmarkProcess measures the per-record ingest cost: the path every flow
// from every exporter takes before it reaches a batch.
func BenchmarkProcess(b *testing.B) {
	for _, n := range []int{1000, 10000} {
		b.Run(fmt.Sprintf("records=%d", n), func(b *testing.B) {
			p := benchPipeline(b)
			recs := benchRecords(n)
			b.ReportAllocs()
			b.SetBytes(int64(n))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.mu.Lock()
				for j := range recs {
					p.process(&recs[j])
				}
				p.batch = p.batch[:0]
				p.mu.Unlock()
			}
		})
	}
}

// BenchmarkProcessSpanning measures records that straddle minute boundaries,
// where one record becomes several rows.
func BenchmarkProcessSpanning(b *testing.B) {
	p := benchPipeline(b)
	recs := benchRecords(1000)
	for i := range recs {
		recs[i].End = recs[i].Start.Add(5 * time.Minute)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.mu.Lock()
		for j := range recs {
			p.process(&recs[j])
		}
		p.batch = p.batch[:0]
		p.mu.Unlock()
	}
}
