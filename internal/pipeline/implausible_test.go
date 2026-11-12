package pipeline

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/store"
)

// A forged export with an impossible sampling rate and frame length must
// neither count nor make the totals fail (it used to overflow every sum).
func TestImplausibleRecordsRefused(t *testing.T) {
	p, st := setup(t)
	now := time.Now().UTC().Truncate(time.Minute)
	ts := now.Add(-10 * time.Minute)
	rec := func(bytes uint64, mult float64) flow.Record {
		return flow.Record{Start: ts, End: ts.Add(59 * time.Second), Src: netip.MustParseAddr("10.1.1.1"), Dst: netip.MustParseAddr("198.51.100.9"),
			SrcPort: 50000, DstPort: 443, Proto: 6, Bytes: bytes, Packets: 1, Mult: mult, SamplingKnown: true, L2: true,
			Exporter: netip.MustParseAddr("10.0.0.1"), Source: flow.SrcSFlow}
	}
	p.Ingest([]flow.Record{rec(78, 100), rec(0xFFFFFFFF, 0xFFFFFFFF), rec(0xFFFFFFFF, 0xFFFFFFFF)})
	p.FlushRows()
	p.FlushRollups()
	if n := p.Implausible.Load(); n != 2 {
		t.Fatalf("implausible %d, want 2", n)
	}
	tot, err := st.Totals(store.Query{From: now.Add(-time.Hour), To: now})
	if err != nil || tot.Wire != 7800 {
		t.Fatalf("totals %+v %v, want wire 7800", tot, err)
	}
	// and a sum beyond 64 bits saturates instead of failing the query
	var got uint64
	err = st.DB.QueryRow(`SELECT CAST(CASE WHEN sum(x) > 18446744073709551615::HUGEINT THEN 18446744073709551615::HUGEINT ELSE sum(x) END AS UBIGINT)
		FROM (VALUES (18446744073709551615::UBIGINT), (5::UBIGINT)) t(x)`).Scan(&got)
	if err != nil || got != 18446744073709551615 {
		t.Fatalf("saturating sum %d %v", got, err)
	}
}
