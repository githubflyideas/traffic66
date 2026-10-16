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
// testClock is the clock these tests build their traffic from: today's
// date at a fixed hour, never time.Now().
//
// The simulator's background traffic follows the time of day, and the
// detection rules see the attack through 1:4096 sampling, so the hour a
// run starts decides how many of the attack's packets are sampled at all.
// The port scan is the thinnest of them: across the day its sample ranges
// from 6 to 27 distinct ports under 32768, against a rule that asks for 5.
// At the 15:00 hour it comes within one port of that threshold, which is
// why this test used to fail about once in a hundred runs — on whichever
// platform happened to start its run in the wrong minute, which looked
// like a platform problem and was not one.
//
// A fixed hour makes every run draw the same traffic, so a failure here
// is a real change in the rules rather than an unlucky minute. 10:00 is a
// busy office hour, so the rules still have to find the attack through a
// day's worth of normal traffic rather than a quiet night, and it leaves
// the port scan 18 distinct ports against the threshold of 5. The date
// stays current so that retention never reaches the data.
func testClock() time.Time {
	return time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
}

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
	now := testClock()
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
	// The port scan is the thinnest step: the rule asks for PortScanSampled
	// distinct ports and 1:4096 sampling throws most of them away. Measure
	// the room in the run that just happened — the finding carries the count
	// the rule itself arrived at — so whoever next changes the simulator or
	// the sampling rate is told in one line instead of learning it months
	// later from a test that fails for reasons that look random.
	//
	// The count has to come from the real run. Reading the attack window out
	// of a freshly built simulator measures something else: its random
	// stream is at a different point after seven hours of background traffic
	// than on the first call, so that number is unrelated to what the rule
	// sees. A check written that way sat comfortably above the threshold
	// while this test was failing, which is worse than no check at all.
	if f := find(fs, KindPortScan, pc, file); f != nil {
		var cfg Config
		cfg.defaults()
		got, want := int(f.Ev["ports"].(float64)), cfg.PortScanSampled
		t.Logf("the port scan reached detection as %d distinct ports, against a threshold of %d", got, want)
		if got < 2*want {
			t.Errorf("the port scan reached detection as only %d distinct ports against a threshold of %d: "+
				"too little room, and this test will start failing for reasons that look random", got, want)
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
