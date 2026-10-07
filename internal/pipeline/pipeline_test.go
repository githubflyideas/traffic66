package pipeline

import (
	"net/netip"
	"strings"
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
	// the newest minute is still arriving and is not drawn (the range ends now)
	drawn := min(len(se.Times), 120)
	if want := float64(wantWire) * float64(drawn) / 120; drawn < 119 || sum-want > 1 || sum-want < -1 {
		t.Fatalf("series sum %f over %d buckets, want %f", sum, len(se.Times), want)
	}
	// Seal everything and query again across hot+parquet. Seal keeps the
	// hour that is still open (with 5 minutes' grace), so seal as of two
	// hours ahead: one hour ahead fails in the first 5 minutes of an hour.
	if err := st.Seal(now.Add(2 * time.Hour)); err != nil {
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

// Ranking by another measure picks a different top, in both the detail
// query and the rollup.
func TestTopNBy(t *testing.T) {
	p, st := setup(t)
	now := time.Now().UTC().Truncate(time.Hour)
	exp := netip.MustParseAddr("10.0.0.1")
	srv := netip.MustParseAddr("198.51.100.9")
	big, small := netip.MustParseAddr("10.1.1.1"), netip.MustParseAddr("10.1.1.2")
	var recs []flow.Record
	for i := 0; i < 8*60; i++ {
		ts := now.Add(-8*time.Hour + time.Duration(i)*time.Minute)
		// big: one flow of 100 full-size packets; small: 5 flows of 100 tiny packets
		recs = append(recs, flow.Record{Start: ts, End: ts, Src: big, Dst: srv, SrcPort: 50000, DstPort: 443, Proto: 6,
			Bytes: 100 * 1500, Packets: 100, Mult: 1, SamplingKnown: true, Exporter: exp, Source: flow.SrcIPFIX})
		for f := 0; f < 5; f++ {
			recs = append(recs, flow.Record{Start: ts, End: ts, Src: small, Dst: srv, SrcPort: uint16(40000 + f), DstPort: 80, Proto: 6,
				Bytes: 100 * 40, Packets: 100, Mult: 1, SamplingKnown: true, Exporter: exp, Source: flow.SrcIPFIX})
		}
	}
	p.Ingest(recs)
	p.FlushRows()
	p.FlushRollups()
	for _, span := range []time.Duration{time.Hour, 8 * time.Hour} { // detail, rollup
		q := store.Query{From: now.Add(-span), To: now}
		for _, tc := range []struct {
			by   string
			asc  bool
			want string
		}{{"wire", false, "10.1.1.1"}, {"pkts", false, "10.1.1.2"}, {"flows", false, "10.1.1.2"}, {"avg", true, "10.1.1.2"}, {"avg", false, "10.1.1.1"}, {"wire", true, "10.1.1.2"}} {
			top, err := st.TopNBy(q, "client", 1, tc.by, tc.asc)
			if err != nil {
				t.Fatal(err)
			}
			if len(top) != 1 || top[0].Key != tc.want {
				t.Errorf("%v by %s asc=%v: %+v want %s", span, tc.by, tc.asc, top, tc.want)
			}
		}
	}
	if _, err := st.TopNBy(store.Query{From: now.Add(-time.Hour), To: now}, "client", 1, "nope"); err == nil {
		t.Error("unknown ranking accepted")
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

// TestLongRangeConsistent checks that for a range served from summaries,
// starting on a whole hour (as the API does), Top-N adds up to the same total
// as the per-minute totals, so every number on a page counts the same window.
func TestLongRangeConsistent(t *testing.T) {
	p, st := setup(t)
	now := time.Now().UTC().Truncate(time.Minute)
	cli := []netip.Addr{netip.MustParseAddr("10.1.1.1"), netip.MustParseAddr("10.1.1.2"), netip.MustParseAddr("10.1.1.3")}
	srv := netip.MustParseAddr("198.51.100.9")
	exp := netip.MustParseAddr("10.0.0.1")
	var recs []flow.Record
	for m := 0; m < 30*60; m += 7 {
		ts := now.Add(-time.Duration(m) * time.Minute)
		c := cli[m%3]
		recs = append(recs, flow.Record{Start: ts, End: ts.Add(50 * time.Second), Src: c, Dst: srv, SrcPort: 50000, DstPort: 443,
			Proto: 6, Bytes: uint64(1000 + m), Packets: 10, Mult: 1, SamplingKnown: true, Exporter: exp, Source: flow.SrcNetFlow9})
	}
	p.Ingest(recs)
	p.FlushRows()
	p.FlushRollups()
	// To is the moment of the query, after the newest record, as in the API.
	q := store.Query{From: now.Add(-24*time.Hour - 17*time.Minute).Truncate(time.Hour), To: now.Add(30 * time.Second)}
	tot, err := st.Totals(q)
	if err != nil {
		t.Fatal(err)
	}
	top, err := st.TopN(q, "client", 66)
	if err != nil {
		t.Fatal(err)
	}
	var sum uint64
	for _, r := range top {
		sum += r.Wire
	}
	if sum != tot.Wire || sum == 0 {
		t.Fatalf("Top-N clients add up to %d, totals say %d", sum, tot.Wire)
	}
}

// TestDedupWindow checks that the dedup table keeps a fixed window of data
// time: a long flow spread over many minutes must not push the current
// minute out (it did when the table kept the 8 most recently created
// minutes), and minutes older than the window are dropped.
func TestDedupWindow(t *testing.T) {
	p, _ := setup(t)
	now := time.Now().UTC().Truncate(time.Minute)
	a := dkey{netip.MustParseAddr("10.1.1.1"), netip.MustParseAddr("198.51.100.9"), 50000, 443, 6}
	e1 := origin{exporter: netip.MustParseAddr("10.0.0.1")}
	e2 := origin{exporter: netip.MustParseAddr("10.0.0.2")}
	if p.isDup(now, a, e1) {
		t.Fatal("first report is a duplicate")
	}
	// a 60-minute flow from another host touches 60 older minutes
	b := dkey{netip.MustParseAddr("10.1.1.2"), netip.MustParseAddr("198.51.100.9"), 50001, 443, 6}
	for m := 1; m <= 60; m++ {
		p.isDup(now.Add(-time.Duration(m)*time.Minute), b, e1)
	}
	if !p.isDup(now, a, e2) {
		t.Fatal("second observation point not recognised as duplicate after a long flow")
	}
	if p.isDup(now, a, e1) {
		t.Fatal("same observation point counted as duplicate")
	}
	for m := range p.dedup {
		if m.Before(now.Add(-dedupWindow)) {
			t.Fatalf("minute %v outside the window kept", m)
		}
	}
	// the window moves with data time
	p.isDup(now.Add(20*time.Minute), a, e1)
	if len(p.dedup) != 1 {
		t.Fatalf("%d minutes kept after the window moved", len(p.dedup))
	}
}

func TestDedupCap(t *testing.T) {
	p, _ := setup(t)
	old := dedupMaxPerMinute
	dedupMaxPerMinute = 100
	defer func() { dedupMaxPerMinute = old }()
	now := time.Now().UTC().Truncate(time.Minute)
	e1 := origin{exporter: netip.MustParseAddr("10.0.0.1")}
	for i := 0; i < 150; i++ {
		k := dkey{netip.MustParseAddr("10.1.1.1"), netip.MustParseAddr("198.51.100.9"), uint16(1000 + i), 443, 6}
		p.isDup(now, k, e1)
	}
	if n := len(p.dedup[now]); n != 100 {
		t.Fatalf("%d flows kept, cap 100", n)
	}
	if p.DedupFull.Load() != 50 {
		t.Fatalf("DedupFull %d, want 50", p.DedupFull.Load())
	}
}

// Internal hosts are counted on both sides of a conversation, once each.
func TestTotalsInternalHosts(t *testing.T) {
	p, st := setup(t)
	now := time.Now().UTC().Truncate(time.Minute)
	a, b, c := netip.MustParseAddr("10.1.1.1"), netip.MustParseAddr("10.1.1.2"), netip.MustParseAddr("10.1.1.3")
	exp := netip.MustParseAddr("10.0.0.1")
	var recs []flow.Record
	for _, pr := range [][2]netip.Addr{{a, b}, {a, c}, {b, c}} { // a and b are clients and servers
		recs = append(recs, flow.Record{Start: now.Add(-5 * time.Minute), End: now.Add(-5 * time.Minute), Src: pr[0], Dst: pr[1], SrcPort: 50000, DstPort: 445,
			Proto: 6, Bytes: 1000, Packets: 2, Mult: 1, SamplingKnown: true, Exporter: exp, Source: flow.SrcIPFIX})
	}
	p.Ingest(recs)
	p.FlushRows()
	tot, err := st.Totals(store.Query{From: now.Add(-time.Hour), To: now})
	if err != nil || tot.Hosts != 3 || tot.Peers != 0 {
		t.Fatalf("totals %+v %v, want 3 hosts", tot, err)
	}
	tot, err = st.Totals(store.Query{From: now.Add(-time.Hour), To: now, Filters: []store.Filter{{Field: "ip", Value: "10.1.1.1"}}})
	if err != nil || tot.Hosts != 3 {
		t.Fatalf("one host's totals %+v %v, want itself and the 2 it talked to", tot, err)
	}
}

// SeriesBy splits a measure by a dimension, keeps the top values apart and
// sums the rest; RecordCounts counts the records per bucket.
func TestSeriesByAndRecordCounts(t *testing.T) {
	p, st := setup(t)
	now := time.Now().UTC().Truncate(time.Minute)
	exp := netip.MustParseAddr("10.0.0.1")
	srv := netip.MustParseAddr("198.51.100.9")
	var recs []flow.Record
	for i, c := range []string{"10.1.1.1", "10.1.1.2", "10.1.1.3"} {
		for m := 0; m < 3; m++ {
			ts := now.Add(-time.Duration(10+m) * time.Minute)
			recs = append(recs, flow.Record{Start: ts, End: ts, Src: netip.MustParseAddr(c), Dst: srv, SrcPort: 50000, DstPort: 443, Proto: 6,
				Bytes: uint64(1000 * (i + 1)), Packets: uint64(10 * (i + 1)), Mult: 1, SamplingKnown: true, Exporter: exp, InIf: 3, OutIf: 4, Source: flow.SrcIPFIX})
		}
	}
	p.Ingest(recs)
	p.FlushRows()
	q := store.Query{From: now.Add(-time.Hour), To: now}
	se, err := st.SeriesBy(q, "client", "pkts", 2)
	if err != nil {
		t.Fatal(err)
	}
	if se.Distinct != 3 || len(se.Names) != 3 || se.Names[0] != "10.1.1.3" || se.Names[2] != "__other__" || se.Totals[0] != 90 || se.Totals[2] != 30 {
		t.Fatalf("series %v totals %v distinct %d", se.Names, se.Totals, se.Distinct)
	}
	sum := 0.0
	for _, v := range se.Values[0] {
		sum += v * float64(se.Step)
	}
	if sum < 89.9 || sum > 90.1 {
		t.Fatalf("packets per second do not add up to the total: %v", sum)
	}
	for by, want := range map[string]string{"service": "443/tcp\tHTTPS", "if_in": "10.0.0.1/3", "if_out": "10.0.0.1/4"} {
		se, err := st.SeriesBy(q, by, "wire", 8)
		if err != nil || len(se.Names) != 1 || se.Names[0] != want {
			t.Fatalf("%s: %v %v", by, se, err)
		}
	}
	if _, err := st.SeriesBy(q, "nope", "wire", 8); err == nil {
		t.Fatal("unknown series accepted")
	}
	total, hist, err := st.RecordCounts(q)
	if err != nil || total != 9 {
		t.Fatalf("records %d %v", total, err)
	}
	n := 0.0
	for _, v := range hist.Values[0] {
		n += v
	}
	if n != 9 {
		t.Fatalf("histogram holds %v records", n)
	}
	page, err := st.Records(q, 4, 8)
	if err != nil || len(page) != 1 {
		t.Fatalf("last page %d %v", len(page), err)
	}
}

// Charts by client, server and service have no empty buckets between the
// data: a 6-hour range is answered from the flows (5-minute buckets), a
// 24-hour range from the hourly summaries (hourly buckets). Before, both
// showed one spike an hour.
func TestSeriesNoGaps(t *testing.T) {
	p, st := setup(t)
	// not on a 5-minute boundary, as most of the time: rounding the start
	// down makes the range a little longer than 6 hours
	now := time.Now().UTC().Truncate(5 * time.Minute).Add(-3 * time.Minute)
	cli := netip.MustParseAddr("10.1.1.1")
	srv := netip.MustParseAddr("198.51.100.9")
	exp := netip.MustParseAddr("10.0.0.1")
	var recs []flow.Record
	for i := 1; i <= 30*60; i++ {
		ts := now.Add(-time.Duration(i) * time.Minute)
		recs = append(recs, flow.Record{Start: ts, End: ts.Add(59 * time.Second), Src: cli, Dst: srv, SrcPort: 50000, DstPort: 443,
			Proto: 6, Bytes: 1000 * 1482, Packets: 1000, Mult: 1, SamplingKnown: true, Exporter: exp, Source: flow.SrcNetFlow9})
	}
	p.Ingest(recs)
	p.FlushRows()
	p.FlushRollups()
	for _, tc := range []struct {
		span time.Duration
		step int64
	}{{6 * time.Hour, 300}, {24 * time.Hour, 3600}} {
		q := store.Query{From: now.Add(-tc.span), To: now}
		if tc.span > 6*time.Hour {
			q.From = q.From.Truncate(time.Hour)
		}
		for _, by := range []string{"server", "client", "service", "app"} {
			se, err := st.SeriesBy(q, by, "wire", 8)
			if err != nil {
				t.Fatal(err)
			}
			if by != "app" && se.Step != tc.step {
				t.Errorf("%v %s: step %d, want %d", tc.span, by, se.Step, tc.step)
			}
			zero := 0
			for i := range se.Times[:len(se.Times)-1] { // the last bucket is still filling
				sum := 0.0
				for _, v := range se.Values {
					sum += v[i]
				}
				if sum == 0 {
					zero++
				}
			}
			if zero > 0 {
				t.Errorf("%v %s: %d of %d buckets empty", tc.span, by, zero, len(se.Times))
			}
		}
	}
}

// Cleanup deletes flows, summaries and counters before a time, whole hourly
// files at a time; "all" leaves nothing.
func TestPurge(t *testing.T) {
	p, st := setup(t)
	now := time.Now().UTC().Truncate(time.Minute)
	cli := netip.MustParseAddr("10.1.1.1")
	srv := netip.MustParseAddr("198.51.100.9")
	exp := netip.MustParseAddr("10.0.0.1")
	var recs []flow.Record
	for _, age := range []time.Duration{10 * 24 * time.Hour, 3 * 24 * time.Hour, 2 * time.Hour} {
		for i := 0; i < 10; i++ {
			ts := now.Add(-age + time.Duration(i)*time.Minute)
			recs = append(recs, flow.Record{Start: ts, End: ts.Add(59 * time.Second), Src: cli, Dst: srv, SrcPort: 50000, DstPort: 443,
				Proto: 6, Bytes: 1482, Packets: 1, Mult: 1, SamplingKnown: true, Exporter: exp, Source: flow.SrcNetFlow9})
		}
	}
	p.Ingest(recs)
	p.FlushRows()
	p.FlushRollups()
	if err := st.Seal(now); err != nil {
		t.Fatal(err)
	}
	count := func(q string) (n int64) { st.DB.QueryRow(q).Scan(&n); return }
	flows := func() int64 {
		return count(`SELECT count(*) FROM ` + st.Source(now.Add(-30*24*time.Hour), now.Add(time.Hour)))
	}
	if n := flows(); n != 30 {
		t.Fatalf("%d flow rows before", n)
	}
	week := now.Add(-7 * 24 * time.Hour)
	est := st.PurgeSizeBefore(week)
	if est.Rows != 10 || est.Bytes == 0 {
		t.Errorf("estimate %+v", est)
	}
	done, err := st.PurgeBefore(week)
	if err != nil {
		t.Fatal(err)
	}
	if done.Rows != 10 || flows() != 20 {
		t.Errorf("after 7 days: purged %+v, %d left", done, flows())
	}
	if n := count(`SELECT count(*) FROM r_ts WHERE ts < now() - INTERVAL 7 DAY`); n != 0 {
		t.Errorf("%d old summary rows left", n)
	}
	if n := count(`SELECT count(*) FROM r_ts`); n == 0 {
		t.Error("recent summaries deleted")
	}
	if _, err := st.PurgeBefore(time.Time{}); err != nil {
		t.Fatal(err)
	}
	if n := flows(); n != 0 {
		t.Errorf("%d flow rows after deleting all", n)
	}
	for _, tb := range []string{"r_ts", "r_host", "r_dim", "segments", "findings"} {
		if n := count(`SELECT count(*) FROM ` + tb); n != 0 {
			t.Errorf("%s: %d rows after deleting all", tb, n)
		}
	}
}

// Two-ring charts: the top inner values with their outer values, the rest
// as Other; and traffic between named networks with a country and remote
// countries.
func TestRingsAndSegmentCountries(t *testing.T) {
	p, st := setup(t)
	p.Inv.Parse("net 10.1.0.0/16 Tokyo country=JP\n")
	p.ASN.Load(strings.NewReader("198.51.100.0\t198.51.100.255\t64500\tUS\tExample\n"))
	now := time.Now().UTC().Truncate(time.Minute)
	exp := netip.MustParseAddr("10.0.0.1")
	var recs []flow.Record
	add := func(cli, srv string, port uint16, bytes uint64) {
		recs = append(recs, flow.Record{Start: now.Add(-10 * time.Minute), End: now.Add(-9 * time.Minute), Src: netip.MustParseAddr(cli), Dst: netip.MustParseAddr(srv),
			SrcPort: 50000, DstPort: port, Proto: 6, TCPFlags: 0x02, Bytes: bytes, Packets: 1, Mult: 1, SamplingKnown: true, Exporter: exp, Source: flow.SrcNetFlow9})
	}
	add("10.1.1.1", "198.51.100.1", 443, 9000)
	add("10.1.1.2", "198.51.100.1", 443, 3000)
	add("10.1.1.1", "198.51.100.2", 22, 1000)
	p.Ingest(recs)
	p.FlushRows()
	q := store.Query{From: now.Add(-time.Hour), To: now}
	rs, err := st.Rings(q, "server", "client", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs.Inner) != 2 || rs.Inner[0].Key != "198.51.100.1" || rs.Inner[1].Key != "__other__" {
		t.Fatalf("inner %+v", rs.Inner)
	}
	// the top server's two clients: the bigger, and the rest as Other
	var outer []string
	for _, o := range rs.Outer {
		if o.Parent == "198.51.100.1" {
			outer = append(outer, o.Key)
		}
	}
	if len(outer) != 2 || outer[0] != "10.1.1.1" || outer[1] != "__other__" {
		t.Errorf("outer %v", outer)
	}
	if _, err := st.Rings(q, "service", "server", 8, 12); err != nil {
		t.Error(err)
	}
	if _, err := st.Rings(q, "server", "server", 8, 12); err == nil {
		t.Error("same dimension twice accepted")
	}
	sc, err := st.SegmentCountries(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(sc) == 0 || sc[0].Segment != "Tokyo" || sc[0].Country != "US" {
		t.Errorf("segment countries %+v", sc)
	}
	if p.Inv.SegmentCountries()["Tokyo"] != "JP" {
		t.Error("country of the network not parsed")
	}
	if err := p.Inv.Parse("net 10.1.0.0/16 Tokyo country=JPN\n"); err == nil {
		t.Error("three-letter country accepted")
	}
}
