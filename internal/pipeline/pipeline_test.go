package pipeline

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/store"
)

func TestMinuteSlices(t *testing.T) {
	s := time.Date(2026, 1, 1, 10, 0, 30, 0, time.UTC)
	sl := minuteSlices(s, s.Add(90*time.Second))
	if len(sl) != 2 || sl[0].frac != 1.0/3 || sl[1].frac != 2.0/3 {
		t.Fatalf("%+v", sl)
	}
	one := minuteSlices(s, s)
	if len(one) != 1 || one[0].frac != 1 {
		t.Fatalf("%+v", one)
	}
}

func setup(t *testing.T) (*Pipeline, *store.Store) {
	st, err := store.Open(store.Options{Dir: t.TempDir(), RawDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	inv := enrich.NewInventory()
	inv.Parse("net 203.0.113.0/28 DMZ\n")
	asn := enrich.NewASNDB()
	thr := enrich.NewThreats()
	return New(Config{}, st, inv, asn, thr), st
}

func TestIngestQuerySeal(t *testing.T) {
	p, st := setup(t)
	now := time.Now().UTC().Truncate(time.Minute)
	start := now.Add(-2 * time.Hour)
	cli := netip.MustParseAddr("10.1.1.1")
	srv := netip.MustParseAddr("198.51.100.9")
	exp := netip.MustParseAddr("10.0.0.1")
	exp2 := netip.MustParseAddr("10.0.0.2")
	var recs []flow.Record
	// 120 one-minute flows of 1000 packets x 1482 bytes (IP), sampled 1:10 observed.
	for i := 0; i < 120; i++ {
		ts := start.Add(time.Duration(i) * time.Minute)
		recs = append(recs, flow.Record{Start: ts, End: ts.Add(59 * time.Second), Src: cli, Dst: srv, SrcPort: 50000, DstPort: 443,
			Proto: 6, Bytes: 100 * 1482, Packets: 100, Mult: 10, SamplingKnown: true, Exporter: exp, Source: flow.SrcNetFlow9, Direction: 0})
		// same packets reported by a second exporter: duplicate
		recs = append(recs, flow.Record{Start: ts, End: ts.Add(59 * time.Second), Src: cli, Dst: srv, SrcPort: 50000, DstPort: 443,
			Proto: 6, Bytes: 100 * 1482, Packets: 100, Mult: 10, SamplingKnown: true, Exporter: exp2, Source: flow.SrcNetFlow9})
	}
	p.Ingest(recs)
	p.FlushRows()
	p.FlushRollups()
	if p.DupRows.Load() != 120 {
		t.Fatalf("dups %d", p.DupRows.Load())
	}
	q := store.Query{From: start, To: now}
	top, err := st.TopN(q, "client", 66)
	if err != nil {
		t.Fatal(err)
	}
	wantWire := uint64(120 * 1000 * (1482 + 18))
	if len(top) != 1 || top[0].Key != "10.1.1.1" || top[0].Wire != wantWire || top[0].Bytes != 120*1000*1482 || top[0].Flows != 120 {
		t.Fatalf("top %+v want wire %d", top, wantWire)
	}
	tot, err := st.Totals(q)
	if err != nil || tot.Wire != wantWire || tot.Hosts != 1 || tot.Peers != 1 {
		t.Fatalf("totals %+v %v", tot, err)
	}
	se, err := st.SeriesByApp(q, 6)
	if err != nil || len(se.Names) != 1 || se.Names[0] != "HTTPS" {
		t.Fatalf("series %+v %v", se, err)
	}
	var sum float64
	for _, v := range se.Values[0] {
		sum += v * float64(se.Step) / 8
	}
	if d := sum - float64(wantWire); d > 1 || d < -1 {
		t.Fatalf("series sum %f want %d", sum, wantWire)
	}
	// Seal everything older than this hour and query again across hot+parquet.
	if err := st.Seal(now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	top2, err := st.TopN(q, "conv", 66)
	if err != nil || len(top2) != 1 || top2[0].Wire != wantWire || top2[0].Key3 != "443/tcp" {
		t.Fatalf("after seal %+v %v", top2, err)
	}
	if u := st.Usage(); u.Segments != 1 || u.HotRows != 0 {
		t.Fatalf("usage %+v", u)
	}
	// Filters.
	q.Filters = []store.Filter{{Field: "ip", Value: "10.1.1.0/24"}, {Field: "port", Value: "443/tcp"}, {Field: "dir", Value: "outbound"}}
	tot, err = st.Totals(q)
	if err != nil || tot.Wire != wantWire {
		t.Fatalf("filtered %+v %v", tot, err)
	}
	q.Filters = []store.Filter{{Field: "app", Value: "HTTPS", Neg: true}}
	tot, _ = st.Totals(q)
	if tot.Wire != 0 {
		t.Fatalf("negated %+v", tot)
	}
	// Rollup path (no filters, span > 6h).
	q = store.Query{From: now.Add(-24 * time.Hour), To: now}
	top, err = st.TopN(q, "client", 66)
	if err != nil || len(top) != 1 || top[0].Wire != wantWire {
		t.Fatalf("rollup top %+v %v", top, err)
	}
	ports, err := st.TopN(q, "port", 66)
	if err != nil || len(ports) != 1 || ports[0].Key != "443/tcp" || ports[0].Extra != "HTTPS" {
		t.Fatalf("rollup ports %+v %v", ports, err)
	}
	recsOut, err := st.Records(store.Query{From: start, To: now}, 10)
	if err != nil || len(recsOut) != 10 {
		t.Fatalf("records %d %v", len(recsOut), err)
	}
}

func TestL2Source(t *testing.T) {
	p, st := setup(t)
	now := time.Now().UTC().Truncate(time.Minute).Add(-time.Minute)
	p.Ingest([]flow.Record{{Start: now, End: now, Src: netip.MustParseAddr("198.51.100.9"), Dst: netip.MustParseAddr("203.0.113.5"),
		SrcPort: 40000, DstPort: 80, Proto: 6, Bytes: 1518, Packets: 1, L2: true, Mult: 4096, Exporter: netip.MustParseAddr("10.0.0.2"), Source: flow.SrcSFlow}})
	p.FlushRows()
	top, err := st.TopN(store.Query{From: now, To: now.Add(time.Minute)}, "server", 5)
	if err != nil || len(top) != 1 || top[0].Wire != 1518*4096 || !top[0].Int {
		t.Fatalf("%+v %v", top, err)
	}
	tot, _ := st.Totals(store.Query{From: now, To: now.Add(time.Minute), Filters: []store.Filter{{Field: "dir", Value: "inbound"}}})
	if tot.Wire != 1518*4096 {
		t.Fatalf("inbound %+v", tot)
	}
}

// IPv6 addresses have no numeric column; grouping must still keep them apart.
func TestTopNIPv6Groups(t *testing.T) {
	p, st := setup(t)
	now := time.Now().UTC().Truncate(time.Minute)
	exp := netip.MustParseAddr("10.0.0.1")
	srv := netip.MustParseAddr("2001:db8::443")
	var recs []flow.Record
	for i, c := range []string{"2001:db8:1::10", "2001:db8:1::11", "10.1.1.1"} {
		recs = append(recs, flow.Record{Start: now.Add(-10 * time.Minute), End: now.Add(-10 * time.Minute), Src: netip.MustParseAddr(c), Dst: srv,
			SrcPort: 50000, DstPort: 443, Proto: 6, Bytes: uint64(1000 * (i + 1)), Packets: 1, Mult: 1, SamplingKnown: true, Exporter: exp, Source: flow.SrcIPFIX})
	}
	p.Ingest(recs)
	p.FlushRows()
	for _, dim := range []string{"client", "conv"} {
		top, err := st.TopN(store.Query{From: now.Add(-time.Hour), To: now}, dim, 66)
		if err != nil {
			t.Fatal(err)
		}
		if len(top) != 3 || top[0].Key != "10.1.1.1" || top[1].Key != "2001:db8:1::11" || top[2].Key != "2001:db8:1::10" {
			t.Fatalf("%s: %+v", dim, top)
		}
		if dim == "conv" && (top[1].Key2 != "2001:db8::443" || top[1].Key3 != "443/tcp") {
			t.Fatalf("conv keys %+v", top[1])
		}
	}
}

// Detail older than the retention is removed from disk and queries.
func TestRetention(t *testing.T) {
	st, err := store.Open(store.Options{Dir: t.TempDir(), RawDays: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	p := New(Config{}, st, enrich.NewInventory(), enrich.NewASNDB(), enrich.NewThreats())
	now := time.Now().UTC().Truncate(time.Hour)
	exp := netip.MustParseAddr("10.0.0.1")
	rec := func(at time.Time) flow.Record {
		return flow.Record{Start: at, End: at, Src: netip.MustParseAddr("10.1.1.1"), Dst: netip.MustParseAddr("198.51.100.9"),
			SrcPort: 50000, DstPort: 443, Proto: 6, Bytes: 1000, Packets: 1, Mult: 1, SamplingKnown: true, Exporter: exp, Source: flow.SrcIPFIX}
	}
	p.Ingest([]flow.Record{rec(now.Add(-72 * time.Hour)), rec(now.Add(-2 * time.Hour))})
	p.FlushRows()
	if err := st.Seal(now); err != nil {
		t.Fatal(err)
	}
	if u := st.Usage(); u.Segments != 1 {
		t.Fatalf("segments before retention: %+v", u)
	}
	if err := st.Retain(now); err != nil {
		t.Fatal(err)
	}
	all := store.Query{From: now.Add(-100 * time.Hour), To: now, Filters: []store.Filter{{Field: "exporter", Value: exp.String()}}}
	tot, err := st.Totals(all)
	if err != nil {
		t.Fatal(err)
	}
	// Both rows were sealed into one file whose newest row is recent, so the
	// file is kept: retention works on whole files.
	if tot.Flows != 2 {
		t.Fatalf("after retention of a mixed file: %+v", tot)
	}
	if err := st.Retain(now.Add(48 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	tot, _ = st.Totals(all)
	if tot.Flows != 0 || st.Usage().Segments != 0 {
		t.Fatalf("expired data still visible: %+v, %+v", tot, st.Usage())
	}
}
