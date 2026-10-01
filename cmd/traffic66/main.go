// Command traffic66 is a flow analytics tool: it collects sFlow, NetFlow
// and IPFIX, stores them in an embedded DuckDB and serves a web UI and a
// terminal UI.
package main

import (
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
	"strings"
	"syscall"
	"time"

	"github.com/githubflyideas/traffic66/internal/api"
	"github.com/githubflyideas/traffic66/internal/auth"
	"github.com/githubflyideas/traffic66/internal/capture"
	"github.com/githubflyideas/traffic66/internal/collector"
	"github.com/githubflyideas/traffic66/internal/dnsres"
	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/pipeline"
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
	args := os.Args[1:]
	cmd := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
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
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		fmt.Printf(usage, version)
		os.Exit(2)
	}
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
	fs.Float64Var(&f.mem, "memory", 0.10, "share of physical memory the database may use")
	fs.IntVar(&f.l2, "l2-overhead", 18, "bytes added per packet to IP-layer counts to match interface counters")
	fs.DurationVar(&f.hold, "sampling-wait", 5*time.Minute, "how long to hold NetFlow/IPFIX records waiting for the sampling rate")
	fs.StringVar(&f.dnsUpstream, "dns-upstream", "", "DNS server for reverse lookups (default: system resolver)")
	fs.Float64Var(&f.dnsRate, "dns-rate", 20, "maximum reverse lookups per second")
	fs.DurationVar(&f.dnsTTL, "dns-cache", 120*time.Second, "how long reverse lookup results are cached")
	fs.BoolVar(&f.noDNS, "no-dns", false, "disable reverse lookups")
	fs.BoolVar(&f.tuiAfter, "tui", false, "also open the terminal UI in this terminal")
	fs.Parse(args)

	if err := os.MkdirAll(f.data, 0o755); err != nil {
		log.Fatalf("cannot create the data directory: %v; choose one with -data", err)
	}
	if demo {
		prepareDemo(f.data)
	}
	st, err := store.Open(store.Options{Dir: f.data, MemoryFraction: f.mem, RawDays: f.retention})
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	invPath := f.inventory
	if invPath == "" {
		invPath = filepath.Join(f.data, "inventory.txt")
	}
	inv, err := enrich.LoadInventory(invPath)
	if err != nil {
		log.Fatalf("inventory %s: %v", invPath, err)
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
	} else {
		log.Printf("no ASN/country table; put one at %s to see countries and networks", filepath.Join(f.data, "asn.tsv.gz"))
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
	col.NF.SetUnsampled(inv.Unsampled())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if demo {
		backfillDemo(pipe, st)
	}
	go pipe.Run(ctx)

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
	go poller.Run(ctx)

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
	srv := &api.Server{Store: st, Pipe: pipe, Col: col, Inv: inv, ASN: asn, Thr: thr, DNS: dns, Static: web.FS(), Version: version,
		Demo: demo, Check: checker.Check, LocalTok: tok, DataDir: f.data, Started: time.Now()}
	srv.SNMP = poller.Status
	srv.Capture = func() []api.CaptureInfo {
		var out []api.CaptureInfo
		for _, c := range caps {
			out = append(out, api.CaptureInfo{Iface: c.Iface, Method: c.Method, Packets: c.Packets.Load(), Dropped: c.Dropped.Load(), Flows: c.Active(), Err: c.Err()})
		}
		return out
	}
	ln, err := net.Listen("tcp", f.addr)
	if err != nil {
		log.Fatalf("web: %v", err)
	}
	hs := &http.Server{Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go hs.Serve(ln)
	log.Printf("web UI: http://%s", displayAddr(ln.Addr()))

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

func backfillDemo(pipe *pipeline.Pipeline, st *store.Store) {
	var n int64
	st.DB.QueryRow(`SELECT (SELECT count(*) FROM hot) + (SELECT count(*) FROM segments)`).Scan(&n)
	if n > 0 {
		log.Printf("demo: existing data found, skipping backfill")
		return
	}
	now := time.Now()
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
	s2.Backfill(now.Add(-24*time.Hour), now.Add(-time.Minute), true, func(r []flow.Record, c []flow.IfCounters) {
		emit(r, c)
		i++
		if i%30 == 0 {
			pipe.FlushRows()
		}
	})
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
		log.Fatalf("%s: %v", file, err)
	}
	if len(es) == 0 {
		pw = auth.Generate()
		if err := auth.Set(dir, user, pw); err != nil {
			log.Fatalf("saving the password: %v", err)
		}
		log.Printf("first start: sign in as user %q with password %q", user, pw)
		log.Printf("this password is kept (hashed) in %s; change it with: traffic66 passwd -data %s", file, quoteArg(dir))
	} else {
		var users []string
		for _, e := range es {
			users = append(users, e.User)
		}
		log.Printf("login: user %s, password from %s (change it with: traffic66 passwd)", strings.Join(users, ", "), file)
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
	fs.Parse(args)
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
