package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/store"
)

// An hour of traffic over two interfaces: one the device reports counters
// for, one it does not. The report has to speak for both, and say which
// measurement each row rests on.
func TestReport(t *testing.T) {
	st, err := store.Open(store.Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	to := time.Now().UTC().Truncate(5 * time.Minute)
	from := to.Add(-time.Hour)

	// flows: the counted interface carries twice what the other one does,
	// and one minute in the middle carries ten times its usual minute so
	// that the peak and the percentile cannot be the same number.
	const perMin = 60_000_000 // wire bytes a minute on iface 1
	var rows []store.Row
	for i := 0; i < 60; i++ {
		ts := from.Add(time.Duration(i)*time.Minute + 30*time.Second)
		w := uint64(perMin)
		if i == 30 {
			w *= 10
		}
		rows = append(rows,
			store.Row{TS: ts, Exporter: "10.0.0.1", InIf: 1, OutIf: 9, Wire: w, Bytes: w, Pkts: w / 1000, Flows: 1,
				Client: "10.1.1.5", Server: "203.0.113.9", Cli4: ip4(10, 1, 1, 5), Srv4: ip4(203, 0, 113, 9),
				App: "https", SvcPort: 443, Proto: 6, Dir: store.DirOutbound, PeerCC: "US", CliInt: true},
			store.Row{TS: ts, Exporter: "10.0.0.1", InIf: 2, OutIf: 9, Wire: w / 2, Bytes: w / 2, Pkts: w / 2000, Flows: 1,
				Client: "10.1.1.6", Server: "198.51.100.7", Cli4: ip4(10, 1, 1, 6), Srv4: ip4(198, 51, 100, 7),
				App: "dns", SvcPort: 53, Proto: 17, Dir: store.DirOutbound, PeerCC: "NL", CliInt: true})
	}
	if err := st.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	// counters for iface 1 only: a steady megabyte a second, so 8 Mbit/s
	// against a gigabit interface — 0.8% full.
	for i := 0; i <= 120; i++ { // every 30 s across the hour
		ts := from.Add(time.Duration(i) * 30 * time.Second)
		oct := uint64(i) * 30 * 1_000_000
		if err := st.AppendCounter(ts, "10.0.0.1", 1, 1e9, oct, oct/2, 0, 0); err != nil {
			t.Fatal(err)
		}
	}
	st.SaveFinding(store.Finding{Kind: "scan", Sev: 3, Src: "10.1.1.5", First: from.Add(10 * time.Minute), Last: from.Add(20 * time.Minute)}, 30*time.Minute)
	st.SaveFinding(store.Finding{Kind: "threat", Sev: 1, Src: "10.1.1.6", Dst: "203.0.113.9", First: from.Add(time.Minute), Last: to}, 30*time.Minute)

	s := &Server{Store: st}
	get := func(extra string) (int, reportOut) {
		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/report?from=%d&to=%d%s", from.UnixMilli(), to.UnixMilli(), extra)
		s.report(w, httptest.NewRequest("GET", url, nil))
		var out reportOut
		json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	code, out := get("")
	if code != 200 {
		t.Fatalf("%d", code)
	}
	if out.From != from.UnixMilli() || out.To != to.UnixMilli() || out.SpanS != 3600 {
		t.Errorf("period %d..%d span %d", out.From, out.To, out.SpanS)
	}
	// 60 minutes on iface 1 plus half as much on iface 2, with one minute
	// at ten times the rate on each
	wantWire := uint64(perMin*69) + uint64(perMin*69)/2
	if out.Totals.Wire != wantWire {
		t.Errorf("wire %d, want %d", out.Totals.Wire, wantWire)
	}
	if want := float64(wantWire) * 8 / 3600; !near(out.AvgBps, want, 0.001) {
		t.Errorf("avg %.0f bits/s, want %.0f", out.AvgBps, want)
	}
	if out.Step != 60 {
		t.Errorf("step %d s, want per-minute buckets", out.Step)
	}
	// the busy minute is the peak; the percentile is not dragged up to it
	if !near(out.PeakBps, float64(perMin)*10*1.5*8/60, 0.02) {
		t.Errorf("peak %.0f bits/s", out.PeakBps)
	}
	if out.P95Bps >= out.PeakBps || out.P95Bps <= out.AvgBps/2 {
		t.Errorf("p95 %.0f against peak %.0f and average %.0f", out.P95Bps, out.PeakBps, out.AvgBps)
	}
	if out.PeakAt < out.From || out.PeakAt >= out.To {
		t.Errorf("peak at %d, outside the period", out.PeakAt)
	}

	byKey := map[string]reportIface{}
	for _, f := range out.Ifaces {
		byKey[ifKey(f.Exporter, f.IfIndex)] = f
	}
	one, ok := byKey["10.0.0.1/1"]
	if !ok {
		t.Fatalf("no row for the counted interface: %+v", out.Ifaces)
	}
	if one.Source != srcCounters {
		t.Errorf("counted interface reported from %s", one.Source)
	}
	if !near(one.InAvg, 8e6, 0.02) || !near(one.In95, 8e6, 0.02) {
		t.Errorf("counters: avg %.0f, p95 %.0f bits/s, want 8M", one.InAvg, one.In95)
	}
	if one.Speed != 1e9 || !near(one.Util95, 0.8, 0.05) {
		t.Errorf("speed %d, 95th percentile utilisation %.3f%%, want 0.8", one.Speed, one.Util95)
	}
	if one.Coverage < 0.9 {
		t.Errorf("coverage %.2f", one.Coverage)
	}
	two, ok := byKey["10.0.0.1/2"]
	if !ok {
		t.Fatalf("no row for the interface without counters: %+v", out.Ifaces)
	}
	if two.Source != srcFlows {
		t.Errorf("interface without counters reported from %s", two.Source)
	}
	if want := float64(perMin*69) / 2 * 8 / 3600; !near(two.InAvg, want, 0.001) {
		t.Errorf("flows: avg %.0f bits/s, want %.0f", two.InAvg, want)
	}
	if two.In95 == 0 || two.InPeak <= two.In95 {
		t.Errorf("flows: p95 %.0f bits/s is not below the peak %.0f", two.In95, two.InPeak)
	}
	// nothing says how fast that interface is, so no utilisation is given
	// rather than a percentage of a guess
	if two.Speed != 0 || two.Util95 != 0 || two.UtilPeak != 0 {
		t.Errorf("utilisation without a known speed: %+v", two)
	}

	if len(out.TopHosts) == 0 || out.TopHosts[0].Key != "10.1.1.5" {
		t.Errorf("top hosts %+v", out.TopHosts)
	}
	if len(out.TopApps) == 0 || out.TopApps[0].Key != "https" {
		t.Errorf("top applications %+v", out.TopApps)
	}
	if len(out.TopCountries) == 0 || out.TopCountries[0].Key != "US" {
		t.Errorf("top countries %+v", out.TopCountries)
	}
	if out.Findings["high"] != 1 || out.Findings["low"] != 1 || out.Findings["medium"] != 0 {
		t.Errorf("findings %v", out.Findings)
	}
	if len(out.Top) != 2 || out.Top[0].Kind != "scan" {
		t.Errorf("top findings %+v", out.Top)
	}
	// an unregistered installation marks its own report
	if out.Notice == "" {
		t.Error("no evaluation notice on an unregistered installation")
	}

	// one interface: the whole report is then about that interface
	code, only := get("&if=10.0.0.1/2")
	if code != 200 {
		t.Fatalf("%d", code)
	}
	if only.Iface != "10.0.0.1/2" || len(only.Ifaces) != 1 || only.Ifaces[0].IfIndex != 2 {
		t.Fatalf("if=: %+v", only.Ifaces)
	}
	if only.Totals.Wire != uint64(perMin*69)/2 {
		t.Errorf("one interface's wire %d", only.Totals.Wire)
	}
	if len(only.TopApps) != 1 || only.TopApps[0].Key != "dns" {
		t.Errorf("one interface's applications %+v", only.TopApps)
	}

	for _, bad := range []string{"&if=eth0", "&if=10.0.0.1/eth0", "&if=hostname/1"} {
		if code, _ := get(bad); code != 400 {
			t.Errorf("%s: %d, want 400", bad, code)
		}
	}
	w := httptest.NewRecorder()
	s.report(w, httptest.NewRequest("GET", "/api/report?from=zzz&to=1", nil))
	if w.Code != 400 {
		t.Errorf("bad period accepted: %d", w.Code)
	}
}

func near(got, want, tol float64) bool {
	if want == 0 {
		return got == 0
	}
	return math.Abs(got-want)/math.Abs(want) <= tol
}

func ip4(a, b, c, d uint32) uint32 { return a<<24 | b<<16 | c<<8 | d }
