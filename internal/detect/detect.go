// Package detect looks through recent flows for things worth a person's
// attention: scans, password guessing, lateral movement inside the network,
// large uploads to new destinations, floods and traffic with addresses on a
// threat list. Each run looks at a short window of stored flows and records
// what it finds as findings (see store.Finding).
//
// The rules work on sampled data (sFlow, sampled NetFlow): counts of
// distinct targets, ports or connections are what was observed, and a rule
// that needs many observations asks for fewer when the data is sampled but
// then also requires each one to look like a single short probe. What
// sampling hides (a scan of a few hosts behind 1:4096 sampling) is not
// reported; the finding says how the data was sampled.
package detect

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/store"
)

// Config holds the thresholds. Zero values take the defaults.
type Config struct {
	Window time.Duration // flows looked at per run (10 min)
	Every  time.Duration // time between runs (5 min)
	Lag    time.Duration // how long to wait for late flow reports (3 min)
	Gap    time.Duration // a sighting within this of the last one extends a finding (30 min)
	Learn  time.Duration // history needed before "new" can be judged (1 day)

	ScanTargets, ScanTargetsSampled int     // distinct hosts on one port (50 / 5)
	PortScanPorts, PortScanSampled  int     // distinct ports on one host (30 / 5)
	ProbeMaxAvg                     float64 // average frame size of probes, bytes (90)
	BruteConns, BruteConnsSampled   int     // connections to one login service (30 / 5)
	BruteMaxAvg                     float64 // average frame size of login attempts (300)
	ExfilBytes                      float64 // upload to one new destination per window (100 MB)
	ExfilRatio                      float64 // upload at least this times the download (3)
	FloodPPS                        float64 // packets per second to one address (20,000)
	FloodMaxAvg                     float64 // average frame size of a flood (120)
	FloodRatio                      float64 // times the address's usual packet rate (10)
	NewServiceClients               int     // a new service used by this many hosts is a new server, not lateral movement (5)
}

func (c *Config) defaults() {
	def := func(p *time.Duration, v time.Duration) {
		if *p <= 0 {
			*p = v
		}
	}
	def(&c.Window, 10*time.Minute)
	def(&c.Every, 5*time.Minute)
	def(&c.Lag, 3*time.Minute)
	def(&c.Gap, 30*time.Minute)
	def(&c.Learn, 24*time.Hour)
	defi := func(p *int, v int) {
		if *p <= 0 {
			*p = v
		}
	}
	defi(&c.ScanTargets, 50)
	defi(&c.ScanTargetsSampled, 5)
	defi(&c.PortScanPorts, 30)
	defi(&c.PortScanSampled, 5)
	defi(&c.BruteConns, 30)
	defi(&c.BruteConnsSampled, 5)
	defi(&c.NewServiceClients, 5)
	deff := func(p *float64, v float64) {
		if *p <= 0 {
			*p = v
		}
	}
	deff(&c.ProbeMaxAvg, 90)
	deff(&c.BruteMaxAvg, 300)
	deff(&c.ExfilBytes, 100e6)
	deff(&c.ExfilRatio, 3)
	deff(&c.FloodPPS, 20000)
	deff(&c.FloodMaxAvg, 120)
	deff(&c.FloodRatio, 10)
}

// Kinds of findings.
const (
	KindScan     = "scan"     // one source, many hosts, one port
	KindPortScan = "portscan" // one source, one host, many ports
	KindBrute    = "brute"    // many short connections to a login service
	KindLateral  = "lateral"  // inside the network, to admin services never offered there before
	KindExfil    = "exfil"    // large upload to a destination never used before
	KindFlood    = "flood"    // packet flood to one address
	KindThreat   = "threat"   // traffic with an address on a threat list
)

// Ports of login services (password guessing).
var loginPorts = []int{21, 22, 23, 445, 1433, 1521, 3306, 3389, 5432, 5900, 5985, 5986, 6379, 27017}

// sessionMinAvg is the average frame size, in bytes, above which traffic to
// an admin service is a session that moved data: probes and failed logins
// are smaller, and only real sessions teach or trigger the lateral rule.
const sessionMinAvg = 250

// Ports of remote administration and file sharing (lateral movement).
var adminPorts = []int{22, 135, 139, 445, 3389, 5900, 5985, 5986}

// Detector runs the rules against a store.
type Detector struct {
	St  *store.Store
	Inv *enrich.Inventory
	Cfg Config

	mu      sync.Mutex
	lastRun time.Time
	lastErr string
}

// New makes a detector with cfg's thresholds (defaults where zero).
func New(st *store.Store, inv *enrich.Inventory, cfg Config) *Detector {
	cfg.defaults()
	return &Detector{St: st, Inv: inv, Cfg: cfg}
}

// Status describes the detector for the UI.
type Status struct {
	LastRun       time.Time `json:"last_run"`
	LearningUntil time.Time `json:"learning_until"` // "new" rules wait until then; zero when done
	Error         string    `json:"error,omitempty"`
}

func (d *Detector) Status() Status {
	d.mu.Lock()
	st := Status{LastRun: d.lastRun, Error: d.lastErr}
	d.mu.Unlock()
	if st.LastRun.IsZero() { // not run since start: the last window checked before
		if t, err := time.Parse(time.RFC3339, d.St.Meta("detect_last")); err == nil {
			st.LastRun = t
		}
	}
	if o := d.oldest(); !o.IsZero() && time.Since(o) < d.Cfg.Learn {
		st.LearningUntil = o.Add(d.Cfg.Learn)
	}
	return st
}

// Loop runs the rules every Cfg.Every until ctx ends, first catching up on
// windows missed while traffic66 was not running (at most a day).
func (d *Detector) Loop(ctx context.Context) {
	d.CatchUp(time.Now().Add(-d.Cfg.Lag))
	t := time.NewTicker(d.Cfg.Every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			d.CatchUp(now.Add(-d.Cfg.Lag))
		}
	}
}

// CatchUp runs every window from the last run (or a day ago) up to end.
func (d *Detector) CatchUp(end time.Time) {
	end = end.UTC().Truncate(time.Minute)
	from := end.Add(-24 * time.Hour)
	if v := d.St.Meta("detect_last"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil && t.After(from) {
			from = t
		}
	}
	for t := from.Add(d.Cfg.Every); !t.After(end); t = t.Add(d.Cfg.Every) {
		if _, err := d.Run(t); err != nil {
			log.Printf("detect: %v", err)
		}
	}
}

// Run applies every rule to the window ending at end and records findings.
// It returns how many new findings were opened.
func (d *Detector) Run(end time.Time) (int, error) {
	end = end.UTC().Truncate(time.Minute)
	from := end.Add(-d.Cfg.Window)
	w := &window{d: d, from: from, to: end, src: d.St.Source(from, end)}
	if err := d.bootstrap(from); err != nil {
		return 0, d.fail(err)
	}
	oldest := d.oldest()
	learned := !oldest.IsZero() && from.Sub(oldest) >= d.Cfg.Learn
	var all []store.Finding
	for _, rule := range []func() ([]store.Finding, error){w.scans, w.portScans, w.brute, w.flood, w.threats} {
		fs, err := rule()
		if err != nil {
			return 0, d.fail(err)
		}
		all = append(all, fs...)
	}
	if learned {
		for _, rule := range []func() ([]store.Finding, error){w.lateral, w.exfil} {
			fs, err := rule()
			if err != nil {
				return 0, d.fail(err)
			}
			all = append(all, fs...)
		}
	}
	created := 0
	for _, f := range all {
		_, c, err := d.St.SaveFinding(f, d.Cfg.Gap)
		if err != nil {
			return created, d.fail(err)
		}
		if c {
			created++
		}
	}
	if err := w.learn(); err != nil {
		return created, d.fail(err)
	}
	d.St.SetMeta("detect_last", end.Format(time.RFC3339))
	d.mu.Lock()
	d.lastRun, d.lastErr = time.Now(), ""
	d.mu.Unlock()
	return created, nil
}

func (d *Detector) fail(err error) error {
	d.mu.Lock()
	d.lastErr = err.Error()
	d.mu.Unlock()
	return err
}

// oldest is the time of the oldest stored flow.
func (d *Detector) oldest() time.Time {
	var a, b sql.NullTime
	d.St.DB.QueryRow(`SELECT min(min_ts) FROM segments`).Scan(&a)
	d.St.DB.QueryRow(`SELECT min(ts) FROM hot`).Scan(&b)
	switch {
	case a.Valid && b.Valid:
		if b.Time.Before(a.Time) {
			return b.Time
		}
		return a.Time
	case a.Valid:
		return a.Time
	}
	return b.Time
}

// bootstrap fills the "seen before" table from the stored history the first
// time the detector runs, so a new installation does not need a day to learn
// what an upgraded one already has.
func (d *Detector) bootstrap(before time.Time) error {
	if d.St.Meta("detect_seen") != "" {
		return nil
	}
	// the last two days with data: enough to know the usual services and
	// destinations, and cheap enough at high flow rates (it reads every
	// stored flow of them)
	var lastT sql.NullTime
	if o := d.oldest(); !o.IsZero() && o.Before(before) {
		d.St.DB.QueryRow(fmt.Sprintf(`SELECT max(ts) FROM %s WHERE ts < ?`, d.St.Source(o, before)), before.UTC()).Scan(&lastT)
	}
	last := before
	if lastT.Valid && lastT.Time.Before(before) {
		last = lastT.Time.Add(time.Minute)
	}
	w := &window{d: d, from: last.Add(-48 * time.Hour), to: before}
	w.src = d.St.Source(w.from, w.to)
	if err := w.learn(); err != nil {
		return err
	}
	return d.St.SetMeta("detect_seen", before.Format(time.RFC3339))
}

// window is one run over [from, to).
type window struct {
	d        *Detector
	from, to time.Time
	src      string
}

func (w *window) base() string { return "ts >= ? AND ts < ? AND NOT dup" }
func (w *window) args(extra ...any) []any {
	return append([]any{w.from, w.to}, extra...)
}

func (w *window) internal(ip string) bool {
	a, err := netip.ParseAddr(ip)
	return err == nil && w.d.Inv.Internal(a)
}

func intList(v []int) string {
	s := make([]string, len(v))
	for i, x := range v {
		s[i] = strconv.Itoa(x)
	}
	return strings.Join(s, ",")
}

func portLabel(port int, proto int) string {
	switch proto {
	case 1, 58:
		return "icmp"
	case 17:
		return fmt.Sprintf("%d/udp", port)
	}
	return fmt.Sprintf("%d/tcp", port)
}

func split(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

// scans: one source probing many hosts on one port (TCP SYN or ICMP echo).
// Probes are small; on sampled data each observed probe must have gone to a
// different host (at most three observed packets per host).
func (w *window) scans() ([]store.Finding, error) {
	c := w.d.Cfg
	q := fmt.Sprintf(`SELECT src, dport, proto, count(DISTINCT dst) AS nd, sum(pkts), sum(wire), max(mult), min(ts), max(ts),
			array_to_string(list_slice(list(DISTINCT dst), 1, 6), ',')
		FROM %s WHERE %s AND proto IN (1, 6, 58)
		GROUP BY src, dport, proto
		HAVING sum(wire) <= sum(pkts) * CASE WHEN proto = 6 THEN %g ELSE 160 END
		   AND count(DISTINCT dst) >= CASE WHEN max(mult) > 1 THEN %d ELSE %d END
		   AND sum(pkts / greatest(mult, 1)) <= 3 * count(DISTINCT dst)
		ORDER BY nd DESC LIMIT 100`, w.src, w.base(), c.ProbeMaxAvg, c.ScanTargetsSampled, c.ScanTargets)
	rows, err := w.d.St.DB.Query(q, w.args()...)
	if err != nil {
		return nil, fmt.Errorf("scan rule: %w", err)
	}
	defer rows.Close()
	var out []store.Finding
	for rows.Next() {
		var src, ex string
		var port, proto, nd int
		var pk, wire uint64
		var mult float64
		var first, last time.Time
		if err := rows.Scan(&src, &port, &proto, &nd, &pk, &wire, &mult, &first, &last, &ex); err != nil {
			return nil, err
		}
		sev := 1
		if w.internal(src) {
			sev = 3
		}
		out = append(out, store.Finding{Kind: KindScan, Sev: sev, Src: src, Port: portLabel(port, proto), First: first, Last: last.Add(time.Minute),
			Ev: map[string]any{"targets": float64(nd), "pkts": float64(pk), "avg": float64(wire) / float64(max(pk, 1)), "sampling": mult, "examples": split(ex)}})
	}
	return out, rows.Err()
}

// portScans: one source probing many ports of one host. Only ports below
// 32768 count, so the replies of a busy server to many client ports do not.
func (w *window) portScans() ([]store.Finding, error) {
	c := w.d.Cfg
	q := fmt.Sprintf(`SELECT src, dst, count(DISTINCT dport) AS np, sum(pkts), sum(wire), max(mult), min(ts), max(ts),
			array_to_string(list_slice(list(DISTINCT dport ORDER BY dport), 1, 8), ',')
		FROM %s WHERE %s AND proto = 6 AND dport < 32768
		GROUP BY src, dst
		HAVING sum(wire) <= sum(pkts) * %g
		   AND count(DISTINCT dport) >= CASE WHEN max(mult) > 1 THEN %d ELSE %d END
		   AND sum(pkts / greatest(mult, 1)) <= 3 * count(DISTINCT dport)
		ORDER BY np DESC LIMIT 100`, w.src, w.base(), c.ProbeMaxAvg, c.PortScanSampled, c.PortScanPorts)
	rows, err := w.d.St.DB.Query(q, w.args()...)
	if err != nil {
		return nil, fmt.Errorf("port scan rule: %w", err)
	}
	defer rows.Close()
	var out []store.Finding
	for rows.Next() {
		var src, dst, ex string
		var np int
		var pk, wire uint64
		var mult float64
		var first, last time.Time
		if err := rows.Scan(&src, &dst, &np, &pk, &wire, &mult, &first, &last, &ex); err != nil {
			return nil, err
		}
		sev := 1
		if w.internal(src) {
			sev = 3
		}
		out = append(out, store.Finding{Kind: KindPortScan, Sev: sev, Src: src, Dst: dst, First: first, Last: last.Add(time.Minute),
			Ev: map[string]any{"ports": float64(np), "pkts": float64(pk), "avg": float64(wire) / float64(max(pk, 1)), "sampling": mult, "examples": split(ex)}})
	}
	return out, rows.Err()
}

// brute: many short connections from one source to one login service. Each
// connection uses a new source port; on sampled data nearly every observed
// packet must come from a different connection, which long sessions do not.
func (w *window) brute() ([]store.Finding, error) {
	c := w.d.Cfg
	q := fmt.Sprintf(`SELECT src, dst, dport, count(DISTINCT sport) AS ns, sum(pkts), sum(wire), max(mult), min(ts), max(ts)
		FROM %s WHERE %s AND proto = 6 AND dport IN (%s)
		GROUP BY src, dst, dport
		HAVING sum(wire) <= sum(pkts) * CASE WHEN max(mult) > 1 THEN least(%g, 200) ELSE %g END
		   AND count(DISTINCT sport) >= CASE WHEN max(mult) > 1 THEN %d ELSE %d END
		   AND (max(mult) <= 1 OR count(DISTINCT sport) >= 0.7 * sum(pkts / greatest(mult, 1)))
		ORDER BY ns DESC LIMIT 100`, w.src, w.base(), intList(loginPorts), c.BruteMaxAvg, c.BruteMaxAvg, c.BruteConnsSampled, c.BruteConns)
	rows, err := w.d.St.DB.Query(q, w.args()...)
	if err != nil {
		return nil, fmt.Errorf("password guessing rule: %w", err)
	}
	defer rows.Close()
	var out []store.Finding
	for rows.Next() {
		var src, dst string
		var port, ns int
		var pk, wire uint64
		var mult float64
		var first, last time.Time
		if err := rows.Scan(&src, &dst, &port, &ns, &pk, &wire, &mult, &first, &last); err != nil {
			return nil, err
		}
		sev := 1
		if w.internal(src) {
			sev = 3
		}
		out = append(out, store.Finding{Kind: KindBrute, Sev: sev, Src: src, Dst: dst, Port: portLabel(port, 6), First: first, Last: last.Add(time.Minute),
			Ev: map[string]any{"conns": float64(ns), "pkts": float64(pk), "avg": float64(wire) / float64(max(pk, 1)), "sampling": mult}})
	}
	return out, rows.Err()
}

// lateral: inside the network, connections to remote administration or file
// sharing on hosts that never offered that service before (or first did in
// the last day). A new service many hosts start using at once is a new
// server, not an attacker, and is only learned.
func (w *window) lateral() ([]store.Finding, error) {
	c := w.d.Cfg
	q := fmt.Sprintf(`SELECT h.client, h.server, h.svc_port, h.w, h.m, h.f, h.l FROM (
			SELECT client, server, svc_port, sum(wire) AS w, max(mult) AS m, min(ts) AS f, max(ts) AS l
			FROM %s WHERE %s AND dir = %d AND proto = 6 AND svc_port IN (%s)
			GROUP BY client, server, svc_port HAVING sum(wire) > %d * sum(pkts)) h
		LEFT JOIN seen s ON s.kind = 'svc' AND s.a = h.server AND s.b = '' AND s.port = h.svc_port
		WHERE s.first_ts IS NULL OR s.first_ts >= ?`, w.src, w.base(), store.DirInternal, intList(adminPorts), sessionMinAvg)
	rows, err := w.d.St.DB.Query(q, w.args(w.from.Add(-24*time.Hour))...)
	if err != nil {
		return nil, fmt.Errorf("lateral movement rule: %w", err)
	}
	defer rows.Close()
	type hit struct {
		client, server string
		port           int
		wire           uint64
		mult           float64
		first, last    time.Time
	}
	var hits []hit
	users := map[string]map[string]bool{} // server/port -> clients
	for rows.Next() {
		var h hit
		if err := rows.Scan(&h.client, &h.server, &h.port, &h.wire, &h.mult, &h.first, &h.last); err != nil {
			return nil, err
		}
		hits = append(hits, h)
		k := h.server + "/" + strconv.Itoa(h.port)
		if users[k] == nil {
			users[k] = map[string]bool{}
		}
		users[k][h.client] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	type agg struct {
		f       store.Finding
		targets []string
		wire    float64
	}
	by := map[string]*agg{}
	for _, h := range hits {
		if len(users[h.server+"/"+strconv.Itoa(h.port)]) >= c.NewServiceClients {
			continue
		}
		k := h.client + "|" + strconv.Itoa(h.port)
		a := by[k]
		if a == nil {
			a = &agg{f: store.Finding{Kind: KindLateral, Sev: 3, Src: h.client, Port: portLabel(h.port, 6), First: h.first, Last: h.last.Add(time.Minute)}}
			by[k] = a
		}
		a.targets = append(a.targets, h.server)
		a.wire += float64(h.wire)
		if h.first.Before(a.f.First) {
			a.f.First = h.first
		}
		if h.last.Add(time.Minute).After(a.f.Last) {
			a.f.Last = h.last.Add(time.Minute)
		}
		a.f.Ev = map[string]any{"sampling": h.mult}
	}
	var out []store.Finding
	for _, a := range by {
		sort.Strings(a.targets)
		a.f.Ev["targets"] = a.targets
		a.f.Ev["hosts"] = float64(len(a.targets))
		a.f.Ev["bytes"] = a.wire
		out = append(out, a.f)
	}
	return out, nil
}

// exfil: an internal host uploading much more than it downloads to an
// external address it had not exchanged data with before the last day.
func (w *window) exfil() ([]store.Finding, error) {
	c := w.d.Cfg
	q := fmt.Sprintf(`SELECT h.client, h.server, h.up, h.down, h.m, h.f, h.l, h.cc, h.org FROM (
			SELECT client, server, coalesce(sum(wire) FILTER (WHERE src = client), 0) AS up,
				coalesce(sum(wire) FILTER (WHERE src = server), 0) AS down, max(mult) AS m, min(ts) AS f, max(ts) AS l,
				any_value(peer_cc) AS cc, any_value(peer_org) AS org
			FROM %s WHERE %s AND dir = %d
			GROUP BY client, server
			HAVING up >= %g AND up >= %g * down) h
		LEFT JOIN seen s ON s.kind = 'up' AND s.a = h.client AND s.b = h.server AND s.port = 0
		WHERE s.first_ts IS NULL OR s.first_ts >= ?`, w.src, w.base(), store.DirOutbound, c.ExfilBytes, c.ExfilRatio)
	rows, err := w.d.St.DB.Query(q, w.args(w.from.Add(-24*time.Hour))...)
	if err != nil {
		return nil, fmt.Errorf("upload rule: %w", err)
	}
	defer rows.Close()
	var out []store.Finding
	for rows.Next() {
		var cl, sv, cc, org string
		var up, down uint64
		var mult float64
		var first, last time.Time
		if err := rows.Scan(&cl, &sv, &up, &down, &mult, &first, &last, &cc, &org); err != nil {
			return nil, err
		}
		out = append(out, store.Finding{Kind: KindExfil, Sev: 3, Src: cl, Dst: sv, First: first, Last: last.Add(time.Minute),
			Ev: map[string]any{"up": float64(up), "down": float64(down), "sampling": mult, "cc": cc, "org": org}})
	}
	return out, rows.Err()
}

// flood: a high packet rate of small packets to one address, well above
// what it usually receives.
func (w *window) flood() ([]store.Finding, error) {
	c := w.d.Cfg
	secs := w.to.Sub(w.from).Seconds()
	// only flows of small packets count, so a flood is seen even when the
	// address also serves normal traffic
	q := fmt.Sprintf(`SELECT dst, sum(pkts), sum(wire), count(DISTINCT src), max(mult), min(ts), max(ts), mode(dport), mode(proto)
		FROM %s WHERE %s AND wire <= pkts * %g GROUP BY dst
		HAVING sum(pkts) >= %g LIMIT 50`, w.src, w.base(), c.FloodMaxAvg, c.FloodPPS*secs)
	rows, err := w.d.St.DB.Query(q, w.args()...)
	if err != nil {
		return nil, fmt.Errorf("flood rule: %w", err)
	}
	type cand struct {
		dst         string
		pk, wire    uint64
		srcs        int
		mult        float64
		first, last time.Time
		port, proto int
	}
	var cs []cand
	for rows.Next() {
		var x cand
		if err := rows.Scan(&x.dst, &x.pk, &x.wire, &x.srcs, &x.mult, &x.first, &x.last, &x.port, &x.proto); err != nil {
			rows.Close()
			return nil, err
		}
		cs = append(cs, x)
	}
	rows.Close()
	var out []store.Finding
	for _, x := range cs {
		// usual rate: packets the address exchanged per hour with data over
		// the previous week, from the hourly summaries
		var pk sql.NullFloat64
		var hours int
		w.d.St.DB.QueryRow(`SELECT sum(pkts), count(DISTINCT ts) FROM r_host WHERE ip = ? AND ts >= ? AND ts < ?`,
			x.dst, w.from.Add(-7*24*time.Hour), w.from.Add(-time.Hour)).Scan(&pk, &hours)
		usual := 0.0
		if hours > 0 {
			usual = pk.Float64 / (float64(hours) * 3600)
		}
		pps := float64(x.pk) / secs
		if pps < c.FloodRatio*usual {
			continue
		}
		out = append(out, store.Finding{Kind: KindFlood, Sev: 2, Dst: x.dst, Port: portLabel(x.port, x.proto), First: x.first, Last: x.last.Add(time.Minute),
			Ev: map[string]any{"pps": pps, "usual": usual, "avg": float64(x.wire) / float64(max(x.pk, 1)), "sources": float64(x.srcs), "sampling": x.mult}})
	}
	return out, nil
}

// threats: traffic with an address on a threat list. An internal host
// connecting to a listed address is serious; a listed address knocking on
// the door from outside is everyday noise.
func (w *window) threats() ([]store.Finding, error) {
	q := fmt.Sprintf(`SELECT client, server, threat, any_value(dir), coalesce(sum(wire) FILTER (WHERE src = client), 0),
			coalesce(sum(wire) FILTER (WHERE src = server), 0), sum(flows), max(mult), min(ts), max(ts)
		FROM %s WHERE %s AND threat <> '' GROUP BY client, server, threat LIMIT 500`, w.src, w.base())
	rows, err := w.d.St.DB.Query(q, w.args()...)
	if err != nil {
		return nil, fmt.Errorf("threat list rule: %w", err)
	}
	defer rows.Close()
	var out []store.Finding
	for rows.Next() {
		var cl, sv, list string
		var dir int
		var up, down, flows uint64
		var mult float64
		var first, last time.Time
		if err := rows.Scan(&cl, &sv, &list, &dir, &up, &down, &flows, &mult, &first, &last); err != nil {
			return nil, err
		}
		sev := 2
		switch dir {
		case store.DirOutbound:
			sev = 3
		case store.DirInbound:
			sev = 1
		}
		out = append(out, store.Finding{Kind: KindThreat, Sev: sev, Src: cl, Dst: sv, Port: list, First: first, Last: last.Add(time.Minute),
			Ev: map[string]any{"list": list, "dir": float64(dir), "up": float64(up), "down": float64(down), "flows": float64(flows), "sampling": mult}})
	}
	return out, rows.Err()
}

// learn records which admin services and upload destinations the window
// shows, for the "never before" rules, and forgets those unused for 30 days.
func (w *window) learn() error {
	db := w.d.St.DB
	if _, err := db.Exec(fmt.Sprintf(`INSERT INTO seen SELECT 'svc', server, '', svc_port, min(ts), max(ts)
		FROM %s WHERE %s AND dir = %d AND proto = 6 AND svc_port IN (%s) GROUP BY server, svc_port HAVING sum(wire) > %d * sum(pkts)
		ON CONFLICT (kind, a, b, port) DO UPDATE SET last_ts = greatest(last_ts, excluded.last_ts)`,
		w.src, w.base(), store.DirInternal, intList(adminPorts), sessionMinAvg), w.args()...); err != nil {
		return fmt.Errorf("learning services: %w", err)
	}
	if _, err := db.Exec(fmt.Sprintf(`INSERT INTO seen SELECT 'up', client, server, 0, min(ts), max(ts)
		FROM %s WHERE %s AND dir = %d GROUP BY client, server HAVING sum(wire) >= 1000000
		ON CONFLICT (kind, a, b, port) DO UPDATE SET last_ts = greatest(last_ts, excluded.last_ts)`,
		w.src, w.base(), store.DirOutbound), w.args()...); err != nil {
		return fmt.Errorf("learning destinations: %w", err)
	}
	_, err := db.Exec(`DELETE FROM seen WHERE last_ts < ?`, w.to.Add(-30*24*time.Hour))
	return err
}
