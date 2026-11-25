package store

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkTopNLongRange is the second phase's comparison: Top over
// seven and thirty days, before and after retention moves the flow
// records' start past the range. Before, a Top of conversations is a
// scan of every record in the range (services and threats with it);
// clients were already answered from the summaries and are the
// control. After, conversations are answered from the conversation
// summaries. A billing of eight addresses over the thirty days is
// measured on the same before and after: a scan of every record,
// then the billing summaries. The fixture is thirty days at a twentieth of the
// production density the XL benchmark uses (25k rows an hour, 18M
// rows), generated set-wise into the hot table, with a pool of
// twenty thousand conversations the rows repeat, as real ones do.
// The seals go a day at a time, as the pipeline's go an hour at a
// time. Run with -benchtime=1x; the setup takes minutes.
func BenchmarkTopNLongRange(b *testing.B) {
	// on the real disk, not the test tmpfs: eighteen million rows do
	// not fit in half a gigabyte
	home, err := os.UserHomeDir()
	if err != nil {
		b.Fatal(err)
	}
	dir, err := os.MkdirTemp(filepath.Join(home, "workspace"), "t66bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)
	s, err := Open(Options{Dir: dir, Threads: 2, RawDays: 7, RollupDays: 180})
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	s.stopFold()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for h := 0; h < 720; h++ {
		ins := fmt.Sprintf(`INSERT INTO hot SELECT
			TIMESTAMP '2026-01-01 00:00:00' + INTERVAL %d HOUR + INTERVAL (r.i %% 60) MINUTE,
			cl.t, sv.t, CAST(10000 + r.i %% 5000 AS USMALLINT), bp.port, bp.proto,
			cl.t, sv.t, cl.n, sv.n, bp.port, bp.app,
			(500 + r.h2 %% 9000) * 3 // 4, 500 + r.h2 %% 9000, 1 + r.h2 %% 90, 1,
			'10.0.0.1', 0, CAST(1 + r.i %% 4 AS UINTEGER), CAST(1 + (r.i >> 9) %% 4 AS UINTEGER), 0, 0, CAST(1 + r.i %% 4 AS UTINYINT),
			sv.cc, CAST(64500 + sv.idx AS UINTEGER), 'Org', 'core', '', 0, 1.0, true, false, true, false
			FROM (SELECT i, hash(i, %d) %% 20000 AS c, hash(i, %d, 7) AS h2 FROM range(25000) AS t(i)) r
			JOIN (SELECT i AS idx, CAST(167772160 + i AS UINTEGER) AS n, concat('10.0.', i // 256, '.', i %% 256) AS t FROM range(3000) AS t(i)) cl ON cl.idx = r.c %% 3000
			JOIN (SELECT i AS idx, CAST(3405774848 + i AS UINTEGER) AS n, concat('203.0.', i // 256, '.', i %% 256) AS t, ['US','DE','JP','NL','FR','GB'][1 + i %% 6] AS cc FROM range(300) AS t(i)) sv ON sv.idx = (r.c * 7) %% 300
			JOIN (VALUES (0,443,6,'HTTPS'),(1,80,6,'HTTP'),(2,53,17,'DNS'),(3,22,6,'SSH'),(4,25,6,'SMTP'),(5,8080,6,'HTTP')) AS bp(idx, port, proto, app) ON bp.idx = r.c %% 6`,
			h, h, h)
		if _, err := s.DB.Exec(ins); err != nil {
			b.Fatalf("generate hour %d: %v", h, err)
		}
	}
	s.wmu.Lock()
	if err := s.rebuildRange(base, base.Add(30*24*time.Hour)); err != nil {
		s.wmu.Unlock()
		b.Fatal(err)
	}
	s.wmu.Unlock()
	for d := 1; d <= 29; d++ {
		if err := s.Seal(base.Add(time.Duration(d) * 24 * time.Hour)); err != nil {
			b.Fatal(err)
		}
	}
	if err := s.foldOnce(base.Add(30 * 24 * time.Hour)); err != nil {
		b.Fatal(err)
	}
	q7 := Query{From: base.Add(20 * 24 * time.Hour), To: base.Add(27 * 24 * time.Hour)}
	q30 := Query{From: base, To: base.Add(30 * 24 * time.Hour)}
	run := func(name string, q Query, dim string) {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rows, err := s.TopNBy(q, dim, 66, "wire")
				if err != nil {
					b.Fatal(err)
				}
				if len(rows) == 0 {
					b.Fatal("no rows")
				}
			}
		})
	}
	// a product line of eight of the fixture's servers, billed over
	// the thirty days: a scan of every record before retention, the
	// billing summaries after it
	line := make([]netip.Addr, 0, 8)
	for i := 1; i <= 8; i++ {
		line = append(line, netip.MustParseAddr(fmt.Sprintf("203.0.0.%d", i)))
	}
	runBill := func(name string) {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				out, err := s.Billing(q30, line)
				if err != nil {
					b.Fatal(err)
				}
				if out.Bytes == 0 {
					b.Fatal("no bytes billed")
				}
			}
		})
	}
	// the old rule: the records still reach back past both ranges
	run("Conv7dDetail", q7, "conv")
	run("Conv30dDetail", q30, "conv")
	run("Client7dRollup", q7, "client")
	run("Client30dRollup", q30, "client")
	runBill("Billing30dDetail")
	if err := s.Retain(base.Add(30 * 24 * time.Hour)); err != nil {
		b.Fatal(err)
	}
	// the new rule: both ranges now start before the records do
	run("Conv7dSummary", q7, "conv")
	run("Conv30dSummary", q30, "conv")
	run("Client7dSummary", q7, "client")
	run("Client30dSummary", q30, "client")
	runBill("Billing30dSummary")
}
