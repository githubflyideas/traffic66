package detect

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/pipeline"
	"github.com/githubflyideas/traffic66/internal/sim"
	"github.com/githubflyideas/traffic66/internal/store"
)

// run simulates the demo network for the last hours (with or without the
// attack), stores it and runs every detection window over it.
func run(t *testing.T, attack bool, hours int) []store.Finding {
	t.Helper()
	st, err := store.Open(store.Options{Dir: t.TempDir(), RawDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	inv := enrich.NewInventory()
	if err := inv.Parse(sim.Inventory()); err != nil {
		t.Fatal(err)
	}
	asn, err := enrich.LoadTable(strings.NewReader(sim.ASNTable()))
	if err != nil {
		t.Fatal(err)
	}
	thr := enrich.NewThreats()
	c2, scan := sim.ThreatList()
	thr.Add("c2", strings.NewReader(c2))
	thr.Add("scanner", strings.NewReader(scan))
	p := pipeline.New(pipeline.Config{}, st, inv, asn, thr)
	now := time.Now().UTC().Truncate(time.Minute)
	// a normal day a week ago, as in the demo: what "before" looks like
	week := 7 * 24 * time.Hour
	old := sim.New(now, 6)
	old.SessionScale = 40
	i := 0
	old.Backfill(now.Add(-24*time.Hour-week), now.Add(-week), false, func(r []flow.Record, _ []flow.IfCounters) {
		p.Ingest(r)
		if i++; i%60 == 0 {
			p.FlushRows()
		}
	})
	s := sim.New(now, 7)
	s.Backfill(now.Add(-time.Duration(hours)*time.Hour), now.Add(-time.Minute), attack, func(r []flow.Record, _ []flow.IfCounters) {
		p.Ingest(r)
		if i++; i%30 == 0 {
			p.FlushRows()
		}
	})
	p.FlushRows()
	p.FlushRollups()
	if err := st.Seal(now); err != nil {
		t.Fatal(err)
	}
	d := New(st, inv, Config{})
	start := time.Now()
	d.CatchUp(now)
	t.Logf("detection over %d h: %v", hours, time.Since(start))
	if e := d.Status().Error; e != "" {
		t.Fatal(e)
	}
	fs, err := st.Findings(store.FindingQuery{From: now.Add(-48 * time.Hour), To: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		t.Logf("%-8s sev %d %s -> %s %s %v hits %d %v", f.Kind, f.Sev, f.Src, f.Dst, f.Port, f.Last.Sub(f.First), f.Hits, f.Ev)
	}
	return fs
}

func find(fs []store.Finding, kind, src, dst string) *store.Finding {
	for i := range fs {
		if fs[i].Kind == kind && (src == "" || fs[i].Src == src) && (dst == "" || fs[i].Dst == dst) {
			return &fs[i]
		}
	}
	return nil
}

// Every step of the demo attack is found, through 1:4096 sFlow sampling.
func TestAttackFound(t *testing.T) {
	if testing.Short() {
		t.Skip("simulates hours of traffic")
	}
	fs := run(t, true, 7)
	const pc, file, c2, web = "10.10.3.27", "10.20.0.15", "203.0.113.200", "203.0.113.5"
	for _, want := range []struct{ kind, src, dst string }{
		{KindScan, pc, ""}, {KindPortScan, pc, file}, {KindBrute, pc, file}, {KindLateral, pc, ""},
		{KindExfil, pc, c2}, {KindThreat, pc, c2}, {KindFlood, "", web},
	} {
		f := find(fs, want.kind, want.src, want.dst)
		if f == nil {
			t.Errorf("no %s finding %s -> %s", want.kind, want.src, want.dst)
			continue
		}
		if want.kind != KindFlood && f.Sev != 3 {
			t.Errorf("%s: severity %d, want 3", want.kind, f.Sev)
		}
	}
	if f := find(fs, KindLateral, pc, ""); f != nil {
		got := f.Ev["targets"].([]any)
		var ts []string
		for _, x := range got {
			ts = append(ts, x.(string))
		}
		for _, w := range []string{"10.10.1.23", "10.10.1.87", "10.10.2.14", "10.10.2.66", "10.10.4.9", "10.10.4.131"} {
			if !slices.Contains(ts, w) {
				t.Errorf("lateral targets %v lack %s", ts, w)
			}
		}
		if len(ts) != 6 {
			t.Errorf("lateral targets %v, want the 6 workstations", ts)
		}
	}
	// one finding per step, not one per detection window
	n := map[string]int{}
	for _, f := range fs {
		if f.Src == pc || f.Kind == KindFlood {
			n[f.Kind+" "+f.Dst+" "+f.Port]++
		}
	}
	for k, c := range n {
		if c > 1 {
			t.Errorf("%d %s findings, want 1", c, k)
		}
	}
}

// Normal traffic raises nothing; only the internet scanner knocking on the
// public website is reported, as low severity.
func TestNormalQuiet(t *testing.T) {
	if testing.Short() {
		t.Skip("simulates hours of traffic")
	}
	fs := run(t, false, 24)
	for _, f := range fs {
		if f.Src == "192.0.2.66" && f.Sev == 1 {
			continue
		}
		t.Errorf("false finding: %s sev %d %s -> %s %s %v", f.Kind, f.Sev, f.Src, f.Dst, f.Port, f.Ev)
	}
}
