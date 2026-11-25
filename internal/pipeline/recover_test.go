package pipeline

import (
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/store"
)

func rollupDump(t *testing.T, st *store.Store) string {
	t.Helper()
	var b strings.Builder
	for _, q := range []string{
		`SELECT ts, app, sum(bytes), sum(wire), sum(pkts), sum(flows) FROM r_ts GROUP BY ALL ORDER BY ALL`,
		`SELECT ts, ip, role, internal, sum(bytes), sum(wire), sum(pkts), sum(flows) FROM r_host GROUP BY ALL ORDER BY ALL`,
		`SELECT ts, dim, val, sum(bytes), sum(wire), sum(pkts), sum(flows) FROM r_dim WHERE dim NOT IN ('obsif', 'obsunk') GROUP BY ALL ORDER BY ALL`,
	} {
		rows, err := st.DB.Query(q)
		if err != nil {
			t.Fatal(err)
		}
		cols, _ := rows.Columns()
		for rows.Next() {
			v := make([]any, len(cols))
			p := make([]any, len(cols))
			for i := range v {
				p[i] = &v[i]
			}
			rows.Scan(p...)
			fmt.Fprintln(&b, v...)
		}
		rows.Close()
	}
	return b.String()
}

// After a stop that did not write the summaries (kill -9), the next start
// rebuilds them from the flow rows: the same summaries a clean run writes.
func TestRecoverRebuildsSummaries(t *testing.T) {
	p, st := setup(t)
	p.Recover(time.Now())
	now := time.Now().UTC().Truncate(time.Minute)
	exp := netip.MustParseAddr("10.0.0.1")
	var recs []flow.Record
	for m := 0; m < 3*60; m += 3 {
		ts := now.Add(-time.Duration(m) * time.Minute)
		src := netip.AddrFrom4([4]byte{10, 1, 1, byte(m % 7)})
		dst := netip.AddrFrom4([4]byte{198, 51, 100, byte(m % 5)})
		if m%4 == 0 {
			dst = netip.MustParseAddr("203.0.113.5")
		}
		recs = append(recs, flow.Record{Start: ts, End: ts.Add(90 * time.Second), Src: src, Dst: dst, SrcPort: 50000, DstPort: uint16(443 + m%3),
			Proto: []uint8{6, 17, 1}[m%3], Bytes: uint64(1000 + m), Packets: 10, Mult: 1, SamplingKnown: true, Exporter: exp, Source: flow.SrcNetFlow9})
	}
	p.Ingest(recs)
	p.FlushRows()
	if err := st.Seal(now); err != nil {
		t.Fatal(err)
	}
	p.FlushRollups()
	want := rollupDump(t, st)

	// a crash: rows written, summaries of the last minute lost
	if _, err := st.DB.Exec(`DELETE FROM r_ts WHERE ts >= ?`, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if rollupDump(t, st) == want {
		t.Fatal("test setup: nothing was lost")
	}
	p2 := New(Config{}, st, p.Inv, p.ASN, p.Thr)
	p2.Recover(time.Now())
	if got := rollupDump(t, st); got != want {
		t.Fatalf("rebuilt summaries differ:\n%s\nwant:\n%s", got, want)
	}
	// a clean stop: nothing is rebuilt (or changed) on the next start
	p2.Stop()
	if !st.WasClean() {
		t.Fatal("not marked clean after Stop")
	}
	New(Config{}, st, p.Inv, p.ASN, p.Thr).Recover(time.Now())
	if got := rollupDump(t, st); got != want {
		t.Fatal("summaries changed after a clean stop")
	}
}
