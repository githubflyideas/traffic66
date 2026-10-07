// Command traffic66 is a flow analytics tool: it collects sFlow, NetFlow
// and IPFIX, stores them in an embedded DuckDB and serves a web UI and a
// terminal UI.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/githubflyideas/traffic66/internal/api"
	"github.com/githubflyideas/traffic66/internal/auth"
	"github.com/githubflyideas/traffic66/internal/capture"
	"github.com/githubflyideas/traffic66/internal/collector"
	"github.com/githubflyideas/traffic66/internal/detect"
	"github.com/githubflyideas/traffic66/internal/dnsres"
	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/license"
	"github.com/githubflyideas/traffic66/internal/pipeline"
	"github.com/githubflyideas/traffic66/internal/sandbox"
	"github.com/githubflyideas/traffic66/internal/sim"
	"github.com/githubflyideas/traffic66/internal/snmp"
	"github.com/githubflyideas/traffic66/internal/store"
	"github.com/githubflyideas/traffic66/internal/tui"
	"github.com/githubflyideas/traffic66/internal/web"
)

var version = "0.1.0-dev"

const usage = `traffic66 %s — flow analytics for sFlow, NetFlow and IPFIX

Usage:
  traffic66 [serve] [flags]     collect flows and serve the web UI (default)
  traffic66 FILE.pcap [...]     analyse up to 3 capture files (pcap, pcapng) in the web UI
  traffic66 demo [flags]        run with a built-in simulated network
  traffic66 tui [flags]         terminal UI (connects to a running traffic66)
  traffic66 simulate -to HOST   send simulated exports to another collector
  traffic66 passwd [flags]      set the login password
  traffic66 interfaces          list interfaces usable for local capture
  traffic66 version

Run "traffic66 <command> -h" for the flags of a command.
`

func main() {
	log.SetFlags(log.LstdFlags)
	args := cleanArgs(os.Args[1:])
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") && !slices.Contains(commands, strings.ToLower(args[0])) && looksLikeCapture(args[0]) {
		runOffline(args)
		return
	}
	cmd := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = strings.ToLower(args[0]), args[1:]
	}
	switch cmd {
	case "serve":
		serve(args, false)
	case "demo":
		serve(args, true)
	case "tui":
		runTUI(args)
	case "simulate":
		simulate(args)
	case "passwd":
		passwd(args)
	case "interfaces":
		list, err := capture.Interfaces()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, i := range list {
			fmt.Println(i)
		}
	case "version", "-version", "--version":
		fmt.Println("traffic66", version)
	case "help", "-h", "--help":
		fmt.Printf(usage, version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q", cmd)
		if s := nearestCommand(cmd); s != "" {
			fmt.Fprintf(os.Stderr, " (did you mean %q?)", s)
		}
		fmt.Fprint(os.Stderr, "\n\n")
		fmt.Printf(usage, version)
		os.Exit(2)
	}
}

var commands = []string{"serve", "demo", "tui", "simulate", "passwd", "interfaces", "version", "help"}

// cleanArgs undoes what copying a command from a web page or chat often adds:
// full-width or non-breaking spaces, quotes around a word, full-width dashes
// and a trailing full stop. Arguments that were only such characters are
// dropped; an empty argument is kept.
func cleanArgs(in []string) []string {
	out := make([]string, 0, len(in))
	for _, orig := range in {
		if orig == "" { // an empty value given on purpose, e.g. -listen ""
			out = append(out, orig)
			continue
		}
		a := strings.TrimFunc(orig, func(r rune) bool {
			return unicode.IsSpace(r) || strings.ContainsRune("'\"`‘’“”「」『』", r)
		})
		a = strings.TrimRight(a, ".。,，;；")
		a = strings.TrimFunc(a, func(r rune) bool {
			return unicode.IsSpace(r) || strings.ContainsRune("'\"`‘’“”「」『』", r)
		})
		if strings.HasPrefix(a, "－") || strings.HasPrefix(a, "—") || strings.HasPrefix(a, "–") {
			_, n := utf8.DecodeRuneInString(a)
			a = "-" + a[n:]
		}
		if a != "" {
			out = append(out, a)
		}
	}
	return out
}

// nearestCommand returns the command within two edits of s, if any.
func nearestCommand(s string) string {
	best, bestD := "", 3
	for _, c := range commands {
		if d := editDistance(s, c); d < bestD {
			best, bestD = c, d
		}
	}
	return best
}

func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

type serveFlags struct {
	addr, data, listen, user, password, asn, inventory string
	dnsUpstream                                        string
	dnsRate                                            float64
	dnsTTL                                             time.Duration
	noDNS                                              bool
	threats                                            multiFlag
	captures                                           multiFlag
	retention                                          int
	mem                                                float64
	l2                                                 int
	hold                                               time.Duration
	tuiAfter                                           bool
}

func serve(args []string, demo bool) {
	doubleClick = startedByDoubleClick()
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	var f serveFlags
	defData := defaultDir("traffic66-data")
	if demo {
		defData = defaultDir("traffic66-demo")
	}
	fs.StringVar(&f.addr, "addr", ":8066", "web UI and API address")
	fs.StringVar(&f.data, "data", defData, "data directory")
	fs.StringVar(&f.listen, "listen", "sflow=:6343,netflow=:2055,ipfix=:4739", "UDP collectors as name=addr, comma separated; every port accepts every protocol; empty disables")
	fs.StringVar(&f.user, "user", "admin", "login user")
	fs.StringVar(&f.password, "password", "", "login password for this run instead of the stored one (or set TRAFFIC66_PASSWORD)")
	fs.StringVar(&f.asn, "asn", "", "ASN/country table (TSV: start end asn cc org, optionally .gz); default <data>/asn.tsv[.gz] if present")
	fs.StringVar(&f.inventory, "inventory", "", "inventory file with network, device and host names (default <data>/inventory.txt)")
	fs.Var(&f.threats, "threat", "threat list as name=path (repeatable); default <data>/threats/*.txt")
	fs.Var(&f.captures, "capture", "capture locally on this interface (repeatable); see 'traffic66 interfaces'")
	fs.IntVar(&f.retention, "retention-days", 30, "days of flow detail to keep (rollups keep 400 days)")
	fs.Float64Var(&f.mem, "memory", 0.10, "share of physical memory for the database cache, and the same again as a soft limit for the rest of the program (each at least 256 MB)")
	fs.IntVar(&f.l2, "l2-overhead", 18, "bytes added per packet to IP-layer counts to match interface counters")
	fs.DurationVar(&f.hold, "sampling-wait", 5*time.Minute, "how long to hold NetFlow/IPFIX records waiting for the sampling rate")
	fs.StringVar(&f.dnsUpstream, "dns-upstream", "", "DNS server for reverse lookups (default: system resolver)")
	fs.Float64Var(&f.dnsRate, "dns-rate", 20, "maximum reverse lookups per second")
	fs.DurationVar(&f.dnsTTL, "dns-cache", 120*time.Second, "how long reverse lookup results are cached")
	fs.BoolVar(&f.noDNS, "no-dns", false, "disable reverse lookups")
	fs.BoolVar(&f.tuiAfter, "tui", false, "also open the terminal UI in this terminal")
	fs.Parse(args)

	if err := os.MkdirAll(f.data, 0o755); err != nil {
		fatalf("cannot create the data directory: %v; choose one with -data", err)
	}
	if demo {
		prepareDemo(f.data)
	}
	setGoMemoryLimit(f.mem)
	st, err := store.Open(store.Options{Dir: f.data, MemoryFraction: f.mem, RawDays: f.retention})
	if err != nil {
		fatalf("store: %v", err)
	}
	defer st.Close()

	invPath := f.inventory
	if invPath == "" {
		invPath = filepath.Join(f.data, "inventory.txt")
	}
	inv, err := enrich.LoadInventory(invPath)
	if err != nil {
		fatalf("inventory %s: %v", invPath, err)
	}
	asn := enrich.NewASNDB()
	asnPath := f.asn
	if asnPath == "" {
		for _, p := range []string{"asn.tsv", "asn.tsv.gz", "ip2asn-combined.tsv.gz"} {
			if _, err := os.Stat(filepath.Join(f.data, p)); err == nil {
				asnPath = filepath.Join(f.data, p)
				break
			}
		}
	}
	if asnPath != "" {
		if err := asn.LoadFile(asnPath); err != nil {
			log.Printf("ASN table %s: %v", asnPath, err)
		} else {
			log.Printf("ASN table: %d ranges from %s", asn.Size(), asnPath)
		}
	}
	for _, err := range asn.LoadMMDBs(f.data) {
		log.Printf("geo database %v", err)
	}
	if err := asn.LoadDBIP(f.data); err != nil {
		log.Printf("%v", err)
	}
	for _, src := range asn.Sources() {
		log.Printf("geo database: %s %s (%s, built %s)", src.Kind, src.File, src.Type, src.Built.Format("2006-01-02"))
	}
	thr := enrich.NewThreats()
	threatFiles := map[string]string{}
	if matches, _ := filepath.Glob(filepath.Join(f.data, "threats", "*.txt")); len(matches) > 0 {
		for _, m := range matches {
			threatFiles[strings.TrimSuffix(filepath.Base(m), ".txt")] = m
		}
	}
	for _, t := range f.threats {
		name, path, ok := strings.Cut(t, "=")
		if !ok {
			name, path = strings.TrimSuffix(filepath.Base(t), filepath.Ext(t)), t
		}
		threatFiles[name] = path
	}
	for name, path := range threatFiles {
		fh, err := os.Open(path)
		if err != nil {
			log.Printf("threat list %s: %v", path, err)
			continue
		}
		n, err := thr.Add(name, fh)
		fh.Close()
		if err != nil {
			log.Printf("threat list %s: %v", path, err)
			continue
		}
		log.Printf("threat list %s: %d entries", name, n)
	}

	pipe := pipeline.New(pipeline.Config{L2Overhead: f.l2}, st, inv, asn, thr)
	col := collector.New(pipe)
	col.NF.HoldFor = f.hold
	col.SetSampling(inv.Unsampled(), inv.Sampling())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	det := detect.New(st, inv, detect.Config{})
	if demo {
		backfillDemo(pipe, st)
	}
	go pipe.Run(ctx)
	go det.Loop(ctx)

	for _, part := range strings.Split(f.listen, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, addr, ok := strings.Cut(part, "=")
		if !ok {
			name, addr = "udp", part
		}
		l, err := col.Listen(addr, name)
		if err != nil {
			log.Printf("collector: %v", err)
			continue
		}
		log.Printf("collector: %s listening on UDP %s", name, addr)
		go col.Serve(ctx, l)
	}
	go func() {
		t := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-t.C:
				col.Tick(now)
			}
		}
	}()

	poller := &snmp.Poller{
		Interval: time.Minute,
		Targets: func() []snmp.Target {
			var out []snmp.Target
			for _, t := range inv.SNMPTargets() {
				out = append(out, snmp.Target{Exporter: t.Exporter, Host: t.Host, Community: t.Community})
			}
			return out
		},
		Emit: pipe.SubmitCounters,
		Names: func(exp netip.Addr, ifs []snmp.Interface) {
			for _, it := range ifs {
				inv.SetAutoIface(exp, it.Index, it.Name, it.Speed)
			}
		},
	}

	var caps []*capture.Capture
	for _, ifc := range f.captures {
		c, err := capture.Start(ctx, ifc, pipe)
		if err != nil {
			log.Printf("capture %s: %v", ifc, err)
			continue
		}
		log.Printf("capture: %s via %s", ifc, c.Method)
		caps = append(caps, c)
	}

	if demo {
		s := sim.New(time.Now(), uint64(time.Now().UnixNano()))
		go s.Live(ctx, func(b []byte, from netip.Addr) { col.Handle(b, from, time.Now()) }, 90*time.Second)
		startDemoAgents(ctx, s)
		log.Printf("demo: simulated exporters are running (core router, switch, firewall, branch router)")
	}
	go func() {
		if demo { // let the simulated devices count a few seconds first
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
			}
		}
		poller.Run(ctx)
	}()

	checker := loginChecker(f.data, f.user, f.password)
	tok := randomHex(24)
	tokPath := filepath.Join(f.data, ".tui-token")
	os.WriteFile(tokPath, []byte(tok), 0o600)

	var dns *dnsres.Resolver
	if f.noDNS {
		dns = dnsres.New(dnsres.Options{PerSecond: 1e-9, Upstream: "127.0.0.1:1"})
	} else {
		dns = dnsres.New(dnsres.Options{Upstream: f.dnsUpstream, PerSecond: f.dnsRate, TTL: f.dnsTTL})
	}
	srv := &api.Server{Store: st, Pipe: pipe, Col: col, Inv: inv, ASN: asn, Thr: thr, DNS: dns, Det: det, Static: web.FS(), Version: version,
		Demo: demo, Check: checker.Check, Exists: checker.Exists, LocalTok: tok, DataDir: f.data, Started: time.Now()}
	srv.SNMP = poller.Status
	licDir := f.data
	if offline != nil {
		// the data directory is temporary: keep the installation in the user's config
		if d, err := os.UserConfigDir(); err == nil && os.MkdirAll(filepath.Join(d, "traffic66"), 0o755) == nil {
			licDir = filepath.Join(d, "traffic66")
		}
	}
	if lic, err := license.Open(licDir); err != nil {
		log.Printf("license: %v", err)
	} else {
		srv.License = lic
		go lic.Loop(ctx.Done(), 4*time.Hour)
	}
	if offline != nil {
		srv.SB = sandbox.NewWith(filepath.Join(f.data, "sandbox"), inv, asn, thr, sandbox.LocalLimits, 0.25)
		srv.Offline = true
		srv.AutoLogin = randomHex(16)
		for _, p := range offline.files {
			if _, err := srv.SB.AddPath(p); err != nil {
				fatalf("%v", err)
			}
		}
	} else {
		srv.SB = sandbox.New(filepath.Join(f.data, "sandbox"), inv, asn, thr)
	}
	defer srv.SB.Close()
	if demo {
		demoSample(f.data, srv.SB)
	}
	srv.Capture = func() []api.CaptureInfo {
		var out []api.CaptureInfo
		for _, c := range caps {
			out = append(out, api.CaptureInfo{Iface: c.Iface, Method: c.Method, Packets: c.Packets.Load(), Dropped: c.Dropped.Load(), Flows: c.Active(), Err: c.Err()})
		}
		return out
	}
	ln, err := net.Listen("tcp", f.addr)
	if err != nil {
		fatalf("web: %v", err)
	}
	hs := &http.Server{Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go hs.Serve(ln)
	log.Printf("web UI: http://%s", displayAddr(ln.Addr()))
	if offline != nil {
		u := "http://" + displayAddr(ln.Addr())
		_, port, _ := net.SplitHostPort(ln.Addr().String())
		fmt.Printf("\ntraffic66 %s: analysing %d capture file(s); nothing is collected or sent\n", version, len(offline.files))
		fmt.Printf("  Web UI    %s  (port %s, this computer only)\n", u, port)
		fmt.Printf("  Sign in   user %s, password %s\n", f.user, f.password)
		fmt.Printf("  Open      %s/auto?t=%s  (signs in once)\n", u, srv.AutoLogin)
		fmt.Printf("  Stop      Ctrl+C; the imported data is deleted, your files are kept\n\n")
		if offline.browser {
			tui.OpenBrowser(u + "/auto?t=" + srv.AutoLogin)
		}
	}
	if doubleClick {
		log.Printf("opening the web UI in your browser; close this window to stop traffic66")
		tui.OpenBrowser("http://" + displayAddr(ln.Addr()))
	}

	if f.tuiAfter {
		err := tui.Run(ctx, tui.Options{URL: "http://" + displayAddr(ln.Addr()), Token: tok})
		stop()
		if err != nil {
			log.Printf("tui: %v", err)
		}
	} else {
		<-ctx.Done()
	}
	log.Printf("shutting down")
	sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	hs.Shutdown(sctx)
	cancel()
	pipe.FlushRows()
	pipe.FlushRollups()
	os.Remove(tokPath)
}

// startDemoAgents serves the simulated devices' interface counters over
// SNMP so the demo reconciles NetFlow and IPFIX the way a real network does.
func startDemoAgents(ctx context.Context, s *sim.Sim) {
	for dev, addr := range sim.SNMPAgents {
		ua, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			continue
		}
		conn, err := net.ListenUDP("udp", ua)
		if err != nil {
			log.Printf("demo: SNMP agent for %s on %s: %v", dev, addr, err)
			continue
		}
		a := &snmp.Agent{Community: "demo", View: func() []snmp.VarBind {
			var rows []snmp.InterfaceRow
			for _, c := range s.DeviceCounters(time.Now(), dev) {
				name, alias := sim.IfAlias(dev, c.IfIndex)
				rows = append(rows, snmp.InterfaceRow{Index: c.IfIndex, Name: name, Alias: alias, SpeedMbps: c.Speed / 1e6,
					InOctets: c.InOctets, OutOctets: c.OutOctets, InPkts: c.InPkts, OutPkts: c.OutPkts})
			}
			return snmp.InterfaceView(rows)
		}}
		go a.Serve(ctx, conn)
	}
}

func displayAddr(a net.Addr) string {
	s := a.String()
	if strings.HasPrefix(s, "[::]:") || strings.HasPrefix(s, "0.0.0.0:") {
		return "127.0.0.1:" + s[strings.LastIndexByte(s, ':')+1:]
	}
	return s
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func prepareDemo(dir string) {
	write := func(name, content string) {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return
		}
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), 0o644)
	}
	write("inventory.txt", sim.Inventory())
	write("asn.tsv", sim.ASNTable())
	c2, scan := sim.ThreatList()
	write("threats/c2.txt", c2)
	write("threats/scanner.txt", scan)
}

// demoSample puts the example capture into the sandbox once, so that offline
// analysis can be tried at once; deleting it keeps it deleted.
func demoSample(dir string, sb *sandbox.Sandbox) {
	mark := filepath.Join(dir, "sandbox-sample")
	if _, err := os.Stat(mark); err == nil {
		return
	}
	var buf bytes.Buffer
	if err := sandbox.Sample(&buf, time.Now().Add(-2*time.Hour).Truncate(time.Minute)); err != nil {
		log.Printf("demo: example capture: %v", err)
		return
	}
	if _, err := sb.Add(sandbox.SampleName, &buf, true); err != nil {
		log.Printf("demo: example capture: %v", err)
		return
	}
	os.WriteFile(mark, nil, 0o644)
}

func backfillDemo(pipe *pipeline.Pipeline, st *store.Store) {
	var n int64
	st.DB.QueryRow(`SELECT (SELECT count(*) FROM hot) + (SELECT count(*) FROM segments)`).Scan(&n)
	now := time.Now()
	if n > 0 {
		// fill the time the demo was stopped, so restarting it leaves no gap
		var last time.Time
		if st.DB.QueryRow(`SELECT max(ts) FROM hot`).Scan(&last) != nil || last.IsZero() || now.Sub(last) > 24*time.Hour {
			log.Printf("demo: existing data found, skipping backfill")
			return
		}
		s := sim.New(now, 67)
		from := last.Add(time.Minute).Truncate(time.Minute)
		for to := now.Truncate(time.Minute); from.Before(to); from, to = to, time.Now().Truncate(time.Minute) {
			s.Backfill(from, to, true, func(r []flow.Record, c []flow.IfCounters) {
				pipe.Ingest(r)
				pipe.IngestCounters(c)
			})
		}
		pipe.FlushRows()
		pipe.FlushRollups()
		log.Printf("demo: filled the time since %s", last.Local().Format("15:04"))
		return
	}
	s := sim.New(now, 66)
	emit := func(recs []flow.Record, ctrs []flow.IfCounters) {
		pipe.Ingest(recs)
		pipe.IngestCounters(ctrs)
	}
	log.Printf("demo: generating one day of history plus the same day last week…")
	week := 7 * 24 * time.Hour
	j := 0
	s.Backfill(now.Add(-24*time.Hour-week), now.Add(-week), false, func(r []flow.Record, c []flow.IfCounters) {
		pipe.Ingest(r)
		j++
		if j%30 == 0 {
			pipe.FlushRows()
		}
	})
	pipe.FlushRows()
	pipe.FlushRollups()
	s2 := sim.New(now, 67)
	i := 0
	feed := func(r []flow.Record, c []flow.IfCounters) {
		emit(r, c)
		i++
		if i%30 == 0 {
			pipe.FlushRows()
		}
	}
	to := now.Truncate(time.Minute)
	s2.Backfill(now.Add(-24*time.Hour), to, true, feed)
	// the minutes that passed while generating, so the charts have no gap
	// where the live exporters take over
	if t := time.Now().Truncate(time.Minute); t.After(to) {
		s2.Backfill(to, t, true, feed)
	}
	pipe.FlushRows()
	pipe.FlushRollups()
	if err := st.Seal(now); err != nil {
		log.Printf("demo: seal: %v", err)
	}
	log.Printf("demo: history ready (%d records, %d rows)", pipe.Records.Load(), pipe.Rows.Load())
}

func runTUI(args []string) {
	fs := flag.NewFlagSet("tui", flag.ExitOnError)
	server := fs.String("server", "http://127.0.0.1:8066", "traffic66 web address")
	data := fs.String("data", "", "data directory of a local traffic66 (for automatic login)")
	user := fs.String("user", "", "login user (not needed for a local traffic66)")
	pass := fs.String("password", "", "login password")
	lang := fs.String("lang", "", "language code, e.g. en, zh, ja, ko (default: from environment)")
	fs.Parse(args)
	opt := tui.Options{URL: strings.TrimRight(*server, "/"), User: *user, Password: *pass, Lang: *lang}
	if *user == "" {
		for _, d := range []string{*data, defaultDir("traffic66-data"), defaultDir("traffic66-demo"), "traffic66-data", "traffic66-demo"} {
			if d == "" {
				continue
			}
			if b, err := os.ReadFile(filepath.Join(d, ".tui-token")); err == nil {
				opt.Token = strings.TrimSpace(string(b))
				break
			}
		}
	}
	if err := tui.Run(context.Background(), opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func simulate(args []string) {
	fs := flag.NewFlagSet("simulate", flag.ExitOnError)
	to := fs.String("to", "127.0.0.1", "collector host; sFlow goes to :6343, NetFlow to :2055, IPFIX to :4739")
	fs.Parse(args)
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		log.Fatal(err)
	}
	dest := func(port int) *net.UDPAddr {
		a, err := net.ResolveUDPAddr("udp", net.JoinHostPort(*to, fmt.Sprint(port)))
		if err != nil {
			log.Fatal(err)
		}
		return a
	}
	sf, nfv, ipf := dest(6343), dest(2055), dest(4739)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	s := sim.New(time.Now(), uint64(time.Now().UnixNano()))
	log.Printf("sending simulated exports to %s (Ctrl+C to stop)", *to)
	s.Live(ctx, func(b []byte, from netip.Addr) {
		d := nfv
		switch {
		case len(b) >= 4 && b[0] == 0 && b[1] == 0 && b[2] == 0 && b[3] == 5:
			d = sf
		case len(b) >= 2 && b[1] == 10:
			d = ipf
		}
		conn.WriteToUDP(b, d)
	}, 90*time.Second)
}

// defaultDir is name next to the executable, so the data directory does not
// depend on the folder traffic66 is started from (a service starts in / or
// C:\Windows\System32).
func defaultDir(name string) string {
	exe, err := os.Executable()
	if err != nil {
		return name
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		exe = r
	}
	return filepath.Join(filepath.Dir(exe), name)
}

// loginChecker decides where the password comes from: -password, then
// TRAFFIC66_PASSWORD, then the password file in the data directory. On the
// very first start it creates the file with a generated password.
func loginChecker(dir, user, flagPw string) *auth.FileChecker {
	pw := flagPw
	if pw == "" {
		pw = os.Getenv("TRAFFIC66_PASSWORD")
	}
	if pw != "" {
		log.Printf("login: user %q with the password given at start", user)
		return auth.NewFileChecker(dir, map[string]string{user: pw})
	}
	file := filepath.Join(dir, auth.FileName)
	es, err := auth.Load(dir)
	if err != nil {
		fatalf("%s: %v", file, err)
	}
	if len(es) == 0 {
		pw = auth.Generate()
		if err := auth.Set(dir, user, pw); err != nil {
			fatalf("saving the password: %v", err)
		}
		log.Printf("first start: sign in as user %q with password %q", user, pw)
		log.Printf("this password is kept (hashed) in %s; change it with: traffic66 passwd -data %s", file, quoteArg(dir))
	} else {
		var users []string
		for _, e := range es {
			users = append(users, e.User)
		}
		log.Printf("login: user %s, password from %s (change it with: traffic66 passwd -data %s)", strings.Join(users, ", "), file, quoteArg(dir))
	}
	return auth.NewFileChecker(dir, nil)
}

func quoteArg(s string) string {
	if strings.ContainsAny(s, " \t\"'") {
		return `"` + s + `"`
	}
	return s
}

func passwd(args []string) {
	fs := flag.NewFlagSet("passwd", flag.ExitOnError)
	data := fs.String("data", defaultDir("traffic66-data"), "data directory")
	user := fs.String("user", "admin", "login user")
	gen := fs.Bool("generate", false, "generate a random password and print it")
	list := fs.Bool("list", false, "list the users")
	del := fs.Bool("delete", false, "delete the user given with -user")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Set, add, list or delete login users. Passwords are stored hashed in
<data directory>/password; a running traffic66 uses changes at the next login.

  traffic66 passwd                      set the password of admin
  traffic66 passwd -user alice          add alice, or change her password
  traffic66 passwd -user alice -delete  delete alice
  traffic66 passwd -list                list the users
  traffic66 passwd -generate            set a random password and print it

Add -data <directory> when traffic66 runs with -data.

`)
		fs.PrintDefaults()
	}
	fs.Parse(args)
	file := filepath.Join(*data, auth.FileName)
	if *list {
		es, err := auth.Load(*data)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if len(es) == 0 {
			fmt.Printf("no users in %s yet; the first start creates admin\n", file)
			return
		}
		for _, e := range es {
			fmt.Println(e.User)
		}
		return
	}
	if *del {
		if err := auth.Delete(*data, *user); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("deleted %s from %s\n", *user, file)
		return
	}
	if err := os.MkdirAll(*data, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	pw := os.Getenv("TRAFFIC66_PASSWORD")
	switch {
	case *gen:
		pw = auth.Generate()
	case pw == "":
		a, err := tui.ReadSecret("new password for " + *user + ": ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if len(a) < 8 {
			fmt.Fprintln(os.Stderr, "use at least 8 characters")
			os.Exit(1)
		}
		b, err := tui.ReadSecret("again: ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if a != b {
			fmt.Fprintln(os.Stderr, "the passwords differ; nothing changed")
			os.Exit(1)
		}
		pw = a
	}
	if err := auth.Set(*data, *user, pw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *gen {
		fmt.Printf("user %s, password %s\n", *user, pw)
	}
	fmt.Printf("saved in %s; a running traffic66 uses it from the next login\n", filepath.Join(*data, auth.FileName))
}

// doubleClick is set when traffic66 was started from Explorer on Windows:
// it then opens the browser, and an error keeps the window open long
// enough to be read instead of closing it at once.
var doubleClick bool

func fatalf(format string, args ...any) {
	log.Printf(format, args...)
	if doubleClick {
		fmt.Fprintln(os.Stderr, "\npress Enter to close this window")
		fmt.Scanln()
	}
	os.Exit(1)
}

// setGoMemoryLimit gives the Go side of the program (decoding, the dedup
// table, row batches) a soft memory limit of the same size as the database's
// share, so that -memory bounds the whole process at about twice that share
// instead of only DuckDB. A soft limit makes the garbage collector work
// harder near it; it cannot stop live data from exceeding it. GOMEMLIMIT in
// the environment takes precedence.
func setGoMemoryLimit(frac float64) {
	if os.Getenv("GOMEMLIMIT") != "" {
		return
	}
	lim := int64(float64(store.TotalMemory()) * frac)
	if lim < 256<<20 {
		lim = 256 << 20
	}
	debug.SetMemoryLimit(lim)
	log.Printf("memory: database %d MB and program %d MB (soft), from -memory %.2f", lim>>20, lim>>20, frac)
}
