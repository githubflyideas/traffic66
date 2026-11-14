package store_test

import (
	"database/sql"
	"fmt"
	"net/netip"
	"sort"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/pipeline"
	"github.com/githubflyideas/traffic66/internal/store"
)

func newPipe(t *testing.T) (*pipeline.Pipeline, *store.Store) {
	st, err := store.Open(store.Options{Dir: t.TempDir(), RawDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	inv := enrich.NewInventory()
	inv.Parse("net 203.0.113.0/28 DMZ\n")
	return pipeline.New(pipeline.Config{}, st, inv, enrich.NewASNDB(), enrich.NewThreats()), st
}

// varied traffic from two devices over ten hours
func traffic(now time.Time) []flow.Record {
	var recs []flow.Record
	addrs := []string{"10.1.1.1", "10.1.1.2", "203.0.113.5", "198.51.100.9", "192.0.2.77", "8.8.8.8"}
	protos := []struct {
		p     uint8
		dport uint16
	}{{6, 443}, {6, 22}, {17, 53}, {1, 0}, {6, 8080}}
	for i := 0; i < 600; i++ {
		ts := now.Add(-10*time.Hour + time.Duration(i)*time.Minute)
		for j, e := range []string{"10.0.0.1", "10.0.0.66"} {
			pr := protos[(i+j)%len(protos)]
			recs = append(recs, flow.Record{Start: ts, End: ts.Add(40 * time.Second),
				Src: netip.MustParseAddr(addrs[(i+j)%len(addrs)]), Dst: netip.MustParseAddr(addrs[(i*7+3+j)%len(addrs)]),
				SrcPort: 40000 + uint16(i%50), DstPort: pr.dport, Proto: pr.p, Bytes: uint64(500 + i%700), Packets: uint64(1 + i%9),
				Mult: 1, SamplingKnown: true, L2: true, Exporter: netip.MustParseAddr(e), Source: flow.SrcSFlow, InIf: uint32(1 + j), OutIf: 3})
		}
	}
	return recs
}

func dump(t *testing.T, db *sql.DB, q string) []string {
	rows, err := db.Query(q)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out []string
	for rows.Next() {
		v := make([]any, len(cols))
		p := make([]any, len(cols))
		for i := range v {
			p[i] = &v[i]
		}
		if err := rows.Scan(p...); err != nil {
			t.Fatal(err)
		}
		out = append(out, fmt.Sprint(v...))
	}
	sort.Strings(out)
	return out
}

func summaries(t *testing.T, st *store.Store) map[string][]string {
	return map[string][]string{
		"r_ts":   dump(t, st.DB, `SELECT ts, app, sum(bytes), sum(wire), sum(pkts), sum(flows) FROM r_ts GROUP BY ALL`),
		"r_host": dump(t, st.DB, `SELECT ts, ip, role, internal, sum(bytes), sum(wire), sum(pkts), sum(flows) FROM r_host GROUP BY ALL`),
		"r_dim":  dump(t, st.DB, `SELECT ts, dim, val, sum(bytes), sum(wire), sum(pkts), sum(flows) FROM r_dim WHERE dim NOT IN ('obsif','obsunk') GROUP BY ALL`),
	}
}

// The rebuild from flow records gives exactly what the pipeline built.
func TestRebuildMatchesPipeline(t *testing.T) {
	p, st := newPipe(t)
	now := time.Now().UTC().Truncate(time.Hour)
	p.Ingest(traffic(now))
	p.FlushRows()
	p.FlushRollups()
	if err := st.Seal(now.Add(2 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	want := summaries(t, st)
	if err := st.RebuildSummaries(now.Add(-11*time.Hour), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got := summaries(t, st)
	for k := range want {
		if len(want[k]) == 0 {
			t.Fatalf("%s empty", k)
		}
		if fmt.Sprint(got[k]) != fmt.Sprint(want[k]) {
			for i := 0; i < len(want[k]) && i < len(got[k]); i++ {
				if want[k][i] != got[k][i] {
					t.Fatalf("%s differs at %d:\n pipeline %s\n rebuilt  %s", k, i, want[k][i], got[k][i])
				}
			}
			t.Fatalf("%s: %d rows from the pipeline, %d rebuilt", k, len(want[k]), len(got[k]))
		}
	}
}

// Deleting one device takes it out of every view.
func TestPurgeExporterEverywhere(t *testing.T) {
	p, st := newPipe(t)
	now := time.Now().UTC().Truncate(time.Hour)
	recs := traffic(now)
	var goodOnly []flow.Record
	for _, r := range recs {
		if r.Exporter.String() == "10.0.0.1" {
			goodOnly = append(goodOnly, r)
		}
	}
	p2, st2 := newPipe(t) // the same traffic without the device, for reference
	p2.Ingest(goodOnly)
	p2.FlushRows()
	p2.FlushRollups()
	p.Ingest(recs)
	p.FlushRows()
	p.FlushRollups()
	for _, s := range []*store.Store{st, st2} {
		if err := s.Seal(now.Add(2 * time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	done, err := st.PurgeExporter("10.0.0.66")
	if err != nil || done.Rows == 0 {
		t.Fatalf("purge %+v %v", done, err)
	}
	for _, q := range []store.Query{
		{From: now.Add(-12 * time.Hour), To: now},
		{From: now.Add(-2 * time.Hour), To: now},
		{From: now.Add(-12 * time.Hour), To: now, Filters: []store.Filter{{Field: "exporter", Value: "10.0.0.66"}}},
	} {
		a, err1 := st.Totals(q)
		b, err2 := st2.Totals(q)
		if err1 != nil || err2 != nil || a.Wire != b.Wire || a.Flows != b.Flows {
			t.Fatalf("%v %v: after purge %+v, without the device %+v (%v %v)", q.To.Sub(q.From), q.Filters, a, b, err1, err2)
		}
	}
	got, want := summaries(t, st), summaries(t, st2)
	for k := range want {
		if fmt.Sprint(got[k]) != fmt.Sprint(want[k]) {
			t.Fatalf("%s differs from the store that never had the device", k)
		}
	}
	exps, _ := st.StoredExporters()
	if len(exps) != 1 || exps[0].Exporter != "10.0.0.1" {
		t.Fatalf("stored exporters %+v", exps)
	}
}

// Flows both devices reported, the deleted one first: the other device's
// copies were duplicates, and must count once it is gone.
func TestPurgeExporterKeepsSharedFlows(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	var first, second []flow.Record
	for _, r := range traffic(now) {
		if r.Exporter.String() == "10.0.0.1" {
			second = append(second, r)
			r.Exporter = netip.MustParseAddr("10.0.0.66")
			first = append(first, r)
		}
	}
	p, st := newPipe(t)
	p.Ingest(first) // the device to delete reports each flow first
	p.Ingest(second)
	p.FlushRows()
	p.FlushRollups()
	p2, st2 := newPipe(t)
	p2.Ingest(second)
	p2.FlushRows()
	p2.FlushRollups()
	for _, s := range []*store.Store{st, st2} {
		if err := s.Seal(now.Add(-3 * time.Hour)); err != nil { // some hours in files, the rest recent rows
			t.Fatal(err)
		}
	}
	if _, err := st.PurgeExporter("10.0.0.66"); err != nil {
		t.Fatal(err)
	}
	for _, q := range []store.Query{{From: now.Add(-12 * time.Hour), To: now}, {From: now.Add(-2 * time.Hour), To: now}} {
		a, _ := st.Totals(q)
		b, _ := st2.Totals(q)
		if a.Wire != b.Wire || a.Wire == 0 {
			t.Fatalf("%v: after purge wire %d, the other device alone %d", q.To.Sub(q.From), a.Wire, b.Wire)
		}
	}
}

// The rebuild after an unclean stop gives exactly what the pipeline built.
func TestRecoveryRebuildMatchesPipeline(t *testing.T) {
	p, st := newPipe(t)
	now := time.Now().UTC().Truncate(time.Hour)
	p.Ingest(traffic(now))
	p.FlushRows()
	p.FlushRollups()
	want := summaries(t, st)
	if _, err := st.RebuildRecentRollups(time.Now()); err != nil {
		t.Fatal(err)
	}
	got := summaries(t, st)
	for k := range want {
		if fmt.Sprint(got[k]) != fmt.Sprint(want[k]) {
			for i := 0; i < len(want[k]) && i < len(got[k]); i++ {
				if want[k][i] != got[k][i] {
					t.Fatalf("%s differs at %d:\n pipeline %s\n rebuilt  %s", k, i, want[k][i], got[k][i])
				}
			}
			t.Fatalf("%s: %d rows from the pipeline, %d rebuilt", k, len(want[k]), len(got[k]))
		}
	}
}
