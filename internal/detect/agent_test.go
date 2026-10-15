package detect

import (
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/store"
)

// agentWindow stores rows and returns a window over them, so each rule can
// be given exactly the traffic it is about. The clock is a fixed hour, not
// time.Now(), for the reason detect_test.go spells out.
func agentWindow(t *testing.T, rows []store.Row, from, to time.Time) *window {
	t.Helper()
	st, err := store.Open(store.Options{Dir: t.TempDir(), RawDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if len(rows) > 0 {
		if err := st.AppendRows(rows); err != nil {
			t.Fatal(err)
		}
	}
	inv := enrich.NewInventory()
	if err := inv.Parse("net 10.0.0.0/8 LAN\n"); err != nil {
		t.Fatal(err)
	}
	d := New(st, inv, Config{})
	return &window{d: d, from: from, to: to, src: st.Source(from, to)}
}

func row(ts time.Time, cli, srv string, sport int, pkts, wire uint64) store.Row {
	return store.Row{TS: ts, Src: cli, Dst: srv, Client: cli, Server: srv,
		SPort: 40000, DPort: uint16(sport), SvcPort: uint16(sport), Proto: 6,
		Exporter: "10.0.0.1", Bytes: wire, Wire: wire, Pkts: pkts, Flows: 1, Mult: 1,
		CliInt: true, SrvInt: true, Dir: store.DirInternal}
}

func kinds(fs []store.Finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.Kind]++
	}
	return m
}

// ---------------------------------------------------------------- fastscan

// One minute, many hosts, many ports, probe-sized packets: a list being
// worked through, not somebody looking at something.
func TestFastScanFires(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	var rows []store.Row
	for h := 0; h < 50; h++ {
		for p := 0; p < 12; p++ {
			rows = append(rows, row(base, "10.1.0.9",
				"10.2."+itoa(h/256)+"."+itoa(h%256), 100+p, 1, 60))
		}
	}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(10*time.Minute))
	fs, err := w.fastScan()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 {
		t.Fatalf("%d findings, want 1", len(fs))
	}
	if fs[0].Src != "10.1.0.9" || fs[0].Sev != 3 {
		t.Errorf("finding %+v", fs[0])
	}
	if fs[0].Ev["targets"].(float64) != 50 || fs[0].Ev["ports"].(float64) != 12 {
		t.Errorf("evidence %+v", fs[0].Ev)
	}
}

// Wide but slow: the same breadth spread over the window is a sweep the
// scan rule reports, not this one. This rule is only about the speed.
func TestFastScanIgnoresSlowSweep(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	var rows []store.Row
	for h := 0; h < 50; h++ {
		for p := 0; p < 12; p++ {
			// one host per minute, so no single minute is wide
			rows = append(rows, row(base.Add(time.Duration(h)*time.Minute), "10.1.0.9",
				"10.2."+itoa(h/256)+"."+itoa(h%256), 100+p, 1, 60))
		}
	}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(60*time.Minute))
	fs, err := w.fastScan()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("%d findings on a slow sweep, want none: %+v", len(fs), fs[0].Ev)
	}
}

// A busy host talking to many peers is not a scan: its packets are not
// probe-sized. Without this the rule would report every proxy and backup
// server in the building.
func TestFastScanIgnoresBusyHost(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	var rows []store.Row
	for h := 0; h < 50; h++ {
		for p := 0; p < 12; p++ {
			rows = append(rows, row(base, "10.1.0.9",
				"10.2."+itoa(h/256)+"."+itoa(h%256), 100+p, 20, 20*1400))
		}
	}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(10*time.Minute))
	fs, err := w.fastScan()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("%d findings on a busy host, want none", len(fs))
	}
}

// ------------------------------------------------------------------ beacon

func extRow(ts time.Time, cli, srv string, wire uint64, org string) store.Row {
	r := row(ts, cli, srv, 443, 4, wire)
	r.SrvInt, r.Dir, r.PeerOrg = false, store.DirOutbound, org
	return r
}

// A check-in every two minutes, carrying almost nothing: a metronome.
func TestBeaconFires(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	var rows []store.Row
	for i := 0; i < 8; i++ {
		rows = append(rows, extRow(base.Add(time.Duration(i*2)*time.Minute), "10.1.0.5", "203.0.113.7", 900, "Example"))
	}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(20*time.Minute))
	fs, err := w.beacon()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 {
		t.Fatalf("%d findings, want 1", len(fs))
	}
	if got := fs[0].Ev["period_s"].(float64); got != 120 {
		t.Errorf("period %v s, want 120", got)
	}
	if got := fs[0].Ev["jitter"].(float64); got != 0 {
		t.Errorf("jitter %v, want 0 for an exact cadence", got)
	}
}

// Somebody reading pages: the same destination, the same small size, but
// at the times a person happens to click. The cadence is the only thing
// separating the two, so this is the test that matters most.
func TestBeaconIgnoresIrregularTraffic(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	offs := []int{0, 1, 5, 6, 7, 13, 14, 19}
	var rows []store.Row
	for _, o := range offs {
		rows = append(rows, extRow(base.Add(time.Duration(o)*time.Minute), "10.1.0.5", "203.0.113.7", 900, "Example"))
	}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(25*time.Minute))
	fs, err := w.beacon()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("%d findings on irregular traffic, want none (jitter %v)", len(fs), fs[0].Ev["jitter"])
	}
}

// A regular cadence that carries real traffic is a download or a sync, not
// a check-in.
func TestBeaconIgnoresRegularButHeavy(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	var rows []store.Row
	for i := 0; i < 8; i++ {
		rows = append(rows, extRow(base.Add(time.Duration(i*2)*time.Minute), "10.1.0.5", "203.0.113.7", 5e6, "Example"))
	}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(20*time.Minute))
	fs, err := w.beacon()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("%d findings on a heavy transfer, want none", len(fs))
	}
}

// cadence itself, away from the database.
func TestCadence(t *testing.T) {
	at := func(mins ...int) []time.Time {
		var out []time.Time
		for _, m := range mins {
			out = append(out, time.Unix(int64(m)*60, 0).UTC())
		}
		return out
	}
	if _, _, ok := cadence(at(0, 5)); ok {
		t.Error("two points were accepted; two points are evenly spaced by definition")
	}
	p, j, ok := cadence(at(0, 5, 10, 15))
	if !ok || p != 5*time.Minute || j != 0 {
		t.Errorf("period %v jitter %v ok %v", p, j, ok)
	}
	if _, j, _ := cadence(at(0, 1, 7, 8)); j < 0.5 {
		t.Errorf("jitter %v on uneven gaps, want a large one", j)
	}
}

// -------------------------------------------------------------------- chain

// A reaches B over SSH, then B reaches C over SSH a minute later.
func TestChainFires(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	rows := []store.Row{
		row(base, "10.1.0.5", "10.1.0.6", 22, 50, 50*400),
		row(base.Add(time.Minute), "10.1.0.6", "10.1.0.7", 22, 50, 50*400),
	}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(10*time.Minute))
	fs, err := w.chain()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 {
		t.Fatalf("%d findings, want 1", len(fs))
	}
	f := fs[0]
	if f.Src != "10.1.0.6" || f.Dst != "10.1.0.7" {
		t.Errorf("want the middle host reported: %+v", f)
	}
	if f.Ev["via"] != "10.1.0.6" {
		t.Errorf("evidence %+v", f.Ev)
	}
	if got := f.Ev["from"].([]string); len(got) != 1 || got[0] != "10.1.0.5" {
		t.Errorf("from %v, want [10.1.0.5]", got)
	}
	if got := f.Ev["to"].([]string); len(got) != 1 || got[0] != "10.1.0.7" {
		t.Errorf("to %v, want [10.1.0.7]", got)
	}
	if f.Ev["in_port"] != "22/tcp" || f.Ev["out_port"] != "22/tcp" {
		t.Errorf("ports %v %v", f.Ev["in_port"], f.Ev["out_port"])
	}
}

// Ordinary file sharing is not a chain. A workstation reaching a file
// server which reaches a backup host is the shape the first version of
// this rule fired on, a hundred times over on a capture with three real
// problems in it. What separates them is that these are services the hosts
// already offered, which is what the seen history records.
func TestChainIgnoresOrdinaryFileSharing(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	var rows []store.Row
	// twenty workstations use the file server, which backs up to two hosts
	for i := 0; i < 20; i++ {
		rows = append(rows, row(base, "10.1.1."+itoa(i), "10.1.0.6", 445, 200, 200*900))
	}
	rows = append(rows, row(base.Add(time.Minute), "10.1.0.6", "10.1.0.7", 445, 200, 200*900))
	rows = append(rows, row(base.Add(time.Minute), "10.1.0.6", "10.1.0.8", 445, 200, 200*900))
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(10*time.Minute))
	fs, err := w.chain()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("%d findings on ordinary file sharing, want none: %+v", len(fs), fs[0].Ev)
	}
}

// One finding per middle host, however many paths run through it: the
// explosion the first version produced was one finding per path.
func TestChainReportsOneFindingPerMiddleHost(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	var rows []store.Row
	// three hosts reach B, and B goes on to three others
	for i := 0; i < 3; i++ {
		rows = append(rows, row(base, "10.1.2."+itoa(i), "10.1.0.6", 22, 50, 50*400))
	}
	for i := 0; i < 3; i++ {
		rows = append(rows, row(base.Add(time.Minute), "10.1.0.6", "10.1.3."+itoa(i), 22, 50, 50*400))
	}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(10*time.Minute))
	fs, err := w.chain()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 {
		t.Fatalf("%d findings for nine paths through one host, want 1", len(fs))
	}
	if got := fs[0].Ev["onward"].(float64); got != 3 {
		t.Errorf("onward %v, want 3", got)
	}
	if got := len(fs[0].Ev["from"].([]string)); got != 3 {
		t.Errorf("from holds %d, want 3", got)
	}
}

// Going back the way it came is not a chain: an administrator on B opening
// a session back to A is ordinary.
func TestChainIgnoresGoingBack(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	rows := []store.Row{
		row(base, "10.1.0.5", "10.1.0.6", 22, 50, 50*400),
		row(base.Add(time.Minute), "10.1.0.6", "10.1.0.5", 22, 50, 50*400),
	}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(10*time.Minute))
	fs, err := w.chain()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("%d findings for a session going back, want none", len(fs))
	}
}

// The second hop long after the first is two separate sessions, which is
// what a person doing their rounds looks like.
func TestChainIgnoresSlowSecondHop(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	rows := []store.Row{
		row(base, "10.1.0.5", "10.1.0.6", 22, 50, 50*400),
		row(base.Add(30*time.Minute), "10.1.0.6", "10.1.0.7", 22, 50, 50*400),
	}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(60*time.Minute))
	fs, err := w.chain()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("%d findings for hops half an hour apart, want none", len(fs))
	}
}

// ---------------------------------------------------------------------- llm

func TestLLMFires(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	rows := []store.Row{extRow(base, "10.1.0.5", "203.0.113.20", 4e6, "OpenAI, L.L.C.")}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(10*time.Minute))
	fs, err := w.llm()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 {
		t.Fatalf("%d findings, want 1", len(fs))
	}
	if fs[0].Sev != 1 {
		t.Errorf("severity %d: this is reported to be known about, not as a fault", fs[0].Sev)
	}
	if fs[0].Ev["org"] != "OpenAI, L.L.C." {
		t.Errorf("evidence %+v", fs[0].Ev)
	}
}

// Another organisation entirely is not reported.
func TestLLMIgnoresOtherOrgs(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	rows := []store.Row{extRow(base, "10.1.0.5", "203.0.113.20", 4e6, "Example Hosting")}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(10*time.Minute))
	fs, err := w.llm()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("%d findings for an unrelated organisation, want none", len(fs))
	}
}

// An empty list turns the rule off rather than matching everything, which
// is the failure that would report every connection on the network.
func TestLLMEmptyListMatchesNothing(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	rows := []store.Row{extRow(base, "10.1.0.5", "203.0.113.20", 4e6, "OpenAI, L.L.C.")}
	w := agentWindow(t, rows, base.Add(-time.Minute), base.Add(10*time.Minute))
	w.d.Cfg.LLMOrgs = nil
	fs, err := w.llm()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("%d findings with no list configured, want none", len(fs))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// ------------------------------------------------------ the port scan's room

// The port-scan rule asks for 5 distinct ports when the data is sampled.
// That is a low bar, and what keeps it from reporting ordinary traffic is
// the two qualifications beside it: only ports below 32768 count, so a
// busy server answering many client ports is not a scan, and the packets
// have to be probe-sized, so a host that really used several services is
// not one either.
//
// This test holds that room open. It is the test the detection rules most
// need and the easiest to lose, because it asserts that nothing happens.
// It went missing once already.
func TestPortScanHasRoom(t *testing.T) {
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	probe := func(rows []store.Row, dst string, port int, pkts, wire uint64) []store.Row {
		return append(rows, row(base, "10.1.0.9", dst, port, pkts, wire))
	}

	// 1. A host that used ten real services: ten ports, but real sessions.
	var real []store.Row
	for _, p := range []int{22, 25, 53, 80, 110, 143, 389, 443, 636, 993} {
		real = probe(real, "10.2.0.1", p, 200, 200*800)
	}
	w := agentWindow(t, real, base.Add(-time.Minute), base.Add(10*time.Minute))
	fs, err := w.portScans()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("a host using ten services was reported as a port scan: %+v", fs[0].Ev)
	}

	// 2. A busy server answering many ephemeral client ports.
	var ephemeral []store.Row
	for i := 0; i < 40; i++ {
		ephemeral = probe(ephemeral, "10.2.0.2", 40000+i, 2, 120)
	}
	w = agentWindow(t, ephemeral, base.Add(-time.Minute), base.Add(10*time.Minute))
	fs, err = w.portScans()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("replies to %d ephemeral ports were reported as a port scan: %+v", 40, fs[0].Ev)
	}

	// 3. And an actual port scan still is one, or the room above would
	// have been bought by turning the rule off.
	var scan []store.Row
	for i := 0; i < 40; i++ {
		scan = probe(scan, "10.2.0.3", 100+i, 1, 60)
	}
	w = agentWindow(t, scan, base.Add(-time.Minute), base.Add(10*time.Minute))
	fs, err = w.portScans()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 {
		t.Fatalf("a real port scan gave %d findings, want 1", len(fs))
	}
	if fs[0].Dst != "10.2.0.3" {
		t.Errorf("finding %+v", fs[0])
	}
}
