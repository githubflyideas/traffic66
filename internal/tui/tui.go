// Package tui is the terminal interface. It talks to a running traffic66
// over its HTTP API and needs no third-party terminal library.
//
// It has three pages: Top (the busiest hosts, servers, applications, ports,
// networks and countries/regions by rate), Interfaces (per device and
// interface rates, utilisation and reconciliation, plus the sampling of each
// device) and Search (flow records for typed filters). Everything else lives
// in the web UI, which "w" opens at the matching view.
package tui

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/githubflyideas/traffic66/internal/web"
)

// Options configure the TUI.
type Options struct {
	URL, Token, User, Password, Lang string
}

// ------------------------------------------------------------ i18n

type dict map[string]string

func loadDict(lang string) (dict, dict) {
	read := func(code string) dict {
		b, err := io.ReadAll(must(web.FS().Open("i18n/" + code + ".json")))
		if err != nil {
			return dict{}
		}
		d := dict{}
		json.Unmarshal(b, &d)
		return d
	}
	return read(lang), read("en")
}

func must(f interface{ Read([]byte) (int, error) }, err error) io.Reader {
	if err != nil {
		return strings.NewReader("{}")
	}
	return f
}

func detectLang(explicit string) string {
	cands := []string{explicit, os.Getenv("LC_ALL"), os.Getenv("LC_MESSAGES"), os.Getenv("LANG"), os.Getenv("LANGUAGE")}
	for _, c := range cands {
		c = strings.ToLower(c)
		if len(c) < 2 {
			continue
		}
		code := c[:2]
		for _, l := range []string{"en", "zh", "hi", "es", "ar", "fr", "bn", "pt", "ru", "id", "ur", "de", "ja", "vi", "ko"} {
			if code == l {
				return l
			}
		}
	}
	return "en"
}

// ------------------------------------------------------------ styles

const (
	reset  = "\x1b[0m"
	bold   = "\x1b[1m"
	dim    = "\x1b[2m"
	rev    = "\x1b[7m"
	red    = "\x1b[31m"
	green  = "\x1b[32m"
	yellow = "\x1b[33m"
	blue   = "\x1b[34m"
	cyan   = "\x1b[36m"
)

// noColor follows https://no-color.org: colours are dropped, while bold,
// dim and reverse video (which carry no colour) stay.
var noColor = os.Getenv("NO_COLOR") != ""

func stripColor(st string) string {
	if !noColor {
		return st
	}
	for _, c := range []string{red, green, yellow, blue, cyan} {
		st = strings.ReplaceAll(st, c, "")
	}
	return st
}

type seg struct{ style, text string }

type line []seg

func (l line) width() int {
	w := 0
	for _, s := range l {
		w += strWidth(s.text)
	}
	return w
}

// render pads or cuts the line to w cells.
func (l line) render(w int, base string) string {
	var b strings.Builder
	used := 0
	for _, s := range l {
		if used >= w {
			break
		}
		t := s.text
		if sw := strWidth(t); used+sw > w {
			t = fit(t, w-used)
		}
		b.WriteString(base + stripColor(s.style) + t + reset)
		used += strWidth(t)
	}
	b.WriteString(base + spaces(w-used) + reset)
	return b.String()
}

// plain is the line's text without styles.
func (l line) plain() string {
	var b strings.Builder
	for _, s := range l {
		b.WriteString(s.text)
	}
	return b.String()
}

// ------------------------------------------------------------ model

type filter struct {
	F   string `json:"f"`
	V   string `json:"v"`
	Neg bool   `json:"neg"`
	lab string
}

type target struct {
	field, value, label string
}

type row struct {
	cells  []line
	target *target
	key    string // stable identity (keeps the selection across refreshes)
	detail []line // shown under the row when it is expanded
}

type column struct {
	head     string
	width    int  // fixed width; 0 = flexible
	right    bool // right-aligned
	optional bool // dropped first on narrow terminals
}

type section struct {
	title line
	cols  []column
	rows  []row
}

type page struct {
	top      []line // fixed lines above the lists
	sections []section
	foot     []line // fixed lines below the lists
	err      string
}

var pageKeys = []string{"top", "ifaces", "search"}
var ranges = []string{"5m", "15m", "1h", "6h", "24h"}
var rangeDur = []time.Duration{5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour, 24 * time.Hour}
var dims = []string{"client", "server", "app", "port", "asn", "country"}
var units = []string{"bps", "bytes", "pps"}
var intervals = []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second}

// narrow is the width below which optional columns are dropped.
const narrow = 80

type app struct {
	opt    Options
	tr, en dict
	cl     *http.Client

	mu       sync.Mutex
	pg       int
	rng      int
	dim      int
	unit     int
	ival     int
	paused   bool
	filters  []filter
	sel      int
	scroll   int
	data     page
	loading  bool
	updated  time.Time
	menu     *target
	menuSel  int
	input    *[]rune // filter being typed
	hist     []string
	histPos  int
	help     bool
	msg      string
	msgUntil time.Time
	names    map[string]string
	ifc      string // exporter/ifindex of the expanded interface
	offset   int    // first record shown on the search page
	selKey   string
	w, h     int
	quit     bool
	refresh  chan struct{}
}

func (a *app) t(k string, kv ...string) string {
	s, ok := a.tr[k]
	if !ok {
		s, ok = a.en[k]
		if !ok {
			s = k
		}
	}
	for i := 0; i+1 < len(kv); i += 2 {
		s = strings.ReplaceAll(s, "{"+kv[i]+"}", kv[i+1])
	}
	return s
}

func newApp(opt Options) *app {
	tr, en := loadDict(detectLang(opt.Lang))
	return &app{opt: opt, tr: tr, en: en, cl: &http.Client{Timeout: 20 * time.Second}, rng: 1, ival: 1,
		names: map[string]string{}, refresh: make(chan struct{}, 1), w: 120, h: 36}
}

// ------------------------------------------------------------ run

// Run starts the terminal UI and blocks until the user quits or ctx ends.
func Run(ctx context.Context, opt Options) error {
	if !isTerminal() {
		return errors.New("traffic66 tui needs an interactive terminal")
	}
	if opt.URL == "" {
		opt.URL = "http://127.0.0.1:8066"
	}
	a := newApp(opt)
	// first contact, before switching the terminal into raw mode
	var st map[string]any
	if err := a.get("status", nil, &st); err != nil {
		if strings.Contains(err.Error(), "401") {
			return errors.New(a.t("tui.login"))
		}
		return errors.New(a.t("tui.error", "url", opt.URL, "e", err.Error()))
	}
	if h, ok := st["hosts"].(map[string]any); ok {
		for k, v := range h {
			a.names[k], _ = v.(string)
		}
	}
	raw, err := makeRaw()
	if err != nil {
		return err
	}
	defer raw.restore()
	out := bufio.NewWriterSize(os.Stdout, 1<<16)
	out.WriteString("\x1b[?1049h\x1b[?25l")
	out.Flush()
	defer func() {
		os.Stdout.WriteString("\x1b[?25h\x1b[?1049l")
	}()

	keys := make(chan string, 16)
	go readKeys(keys)
	a.w, a.h = size()
	go a.fetchLoop(ctx)
	a.kick()

	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for !a.quit {
		a.draw(out)
		select {
		case <-ctx.Done():
			return nil
		case k, ok := <-keys:
			if !ok {
				return nil
			}
			a.key(k)
		case <-tick.C:
			a.w, a.h = size()
		}
	}
	return nil
}

func (a *app) kick() {
	select {
	case a.refresh <- struct{}{}:
	default:
	}
}

func (a *app) fetchLoop(ctx context.Context) {
	for {
		a.mu.Lock()
		iv := intervals[a.ival]
		a.mu.Unlock()
		t := time.NewTimer(iv)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-a.refresh:
			t.Stop()
		case <-t.C:
			a.mu.Lock()
			busy := a.paused || a.menu != nil || a.input != nil || a.help
			a.mu.Unlock()
			if busy {
				continue
			}
		}
		a.reload()
	}
}

// reload fetches the current page once.
func (a *app) reload() {
	a.mu.Lock()
	a.loading = true
	st := a.state()
	a.mu.Unlock()
	p := a.load(st)
	a.mu.Lock()
	if st == a.state() {
		a.data = p
		a.updated = time.Now()
		a.restoreSel()
	}
	a.loading = false
	a.mu.Unlock()
}

// view is everything a page's content depends on.
type view struct {
	pg, rng, dim, unit, offset, w int
	filters, ifc                  string
}

func (a *app) state() view {
	b, _ := json.Marshal(a.filters)
	return view{a.pg, a.rng, a.dim, a.unit, a.offset, a.w, string(b), a.ifc}
}

// restoreSel keeps the selection on the same item after a refresh.
func (a *app) restoreSel() {
	n := a.rowCount()
	if a.selKey != "" {
		i := 0
		for _, s := range a.data.sections {
			for _, r := range s.rows {
				if r.key == a.selKey {
					a.sel = i
					return
				}
				i++
			}
		}
	}
	if a.sel >= n {
		a.sel = max(0, n-1)
	}
	a.rememberSel()
}

func (a *app) rememberSel() {
	if r := a.selRow(); r != nil {
		a.selKey = r.key
	} else {
		a.selKey = ""
	}
}

// ------------------------------------------------------------ http

func (a *app) get(path string, q url.Values, v any) error {
	return a.do("GET", path, q, nil, v)
}

func (a *app) do(method, path string, q url.Values, body io.Reader, v any) error {
	u := a.opt.URL + "/api/" + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return err
	}
	if a.opt.Token != "" {
		req.Header.Set("Authorization", "Bearer "+a.opt.Token)
	} else if a.opt.User != "" {
		req.SetBasicAuth(a.opt.User, a.opt.Password)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := a.cl.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		var e struct{ Error string }
		json.NewDecoder(res.Body).Decode(&e)
		return fmt.Errorf("%d %s", res.StatusCode, e.Error)
	}
	return json.NewDecoder(res.Body).Decode(v)
}

func query(rng int, filters string, extra map[string]string) url.Values {
	q := url.Values{"range": {ranges[rng]}}
	if filters != "" && filters != "null" && filters != "[]" {
		q.Set("f", filters)
	}
	for k, v := range extra {
		q.Set(k, v)
	}
	return q
}

// ------------------------------------------------------------ keys

func readKeys(out chan<- string) {
	r := bufio.NewReader(os.Stdin)
	for {
		b, err := r.ReadByte()
		if err != nil {
			close(out)
			return
		}
		switch {
		case b == 0x1b:
			// escape sequence or a lone Esc
			time.Sleep(15 * time.Millisecond)
			if r.Buffered() == 0 {
				out <- "esc"
				continue
			}
			b2, _ := r.ReadByte()
			if b2 != '[' && b2 != 'O' {
				out <- "esc"
				continue
			}
			seq := ""
			for {
				c, err := r.ReadByte()
				if err != nil {
					break
				}
				seq += string(c)
				if (c >= 'A' && c <= 'Z') || c == '~' || (c >= 'a' && c <= 'z') {
					break
				}
			}
			switch seq {
			case "A":
				out <- "up"
			case "B":
				out <- "down"
			case "C":
				out <- "right"
			case "D":
				out <- "left"
			case "H", "1~":
				out <- "home"
			case "F", "4~":
				out <- "end"
			case "5~":
				out <- "pgup"
			case "6~":
				out <- "pgdn"
			case "3~":
				out <- "delete"
			case "Z":
				out <- "backtab"
			}
		case b == '\r' || b == '\n':
			out <- "enter"
		case b == 0x7f || b == 0x08:
			out <- "backspace"
		case b == '\t':
			out <- "tab"
		case b == 3:
			out <- "ctrl-c"
		case b < 0x20:
		default:
			// UTF-8 rune
			buf := []byte{b}
			for !utf8.FullRune(buf) {
				c, err := r.ReadByte()
				if err != nil {
					break
				}
				buf = append(buf, c)
			}
			out <- string(buf)
		}
	}
}

func (a *app) changed() {
	a.resetSel()
	a.kick()
}

func (a *app) key(k string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if k == "ctrl-c" {
		a.quit = true
		return
	}
	if a.help {
		a.help = false
		return
	}
	if a.input != nil {
		a.inputKey(k)
		return
	}
	if a.menu != nil {
		acts := []string{"only", "not", "records", "copy"}
		switch k {
		case "esc", "q":
			a.menu = nil
		case "up", "k":
			a.menuSel = (a.menuSel + len(acts) - 1) % len(acts)
		case "down", "j", "tab":
			a.menuSel = (a.menuSel + 1) % len(acts)
		case "f":
			a.act("only")
		case "x":
			a.act("not")
		case "r":
			a.act("records")
		case "y":
			a.act("copy")
		case "enter":
			a.act(acts[a.menuSel])
		}
		return
	}
	n := a.rowCount()
	page := pageKeys[a.pg]
	switch k {
	case "q":
		a.quit = true
	case "esc":
		if a.ifc != "" {
			a.ifc = ""
			a.kick()
		}
	case "up", "k":
		if a.sel > 0 {
			a.sel--
		}
	case "down", "j":
		if a.sel < n-1 {
			a.sel++
		}
	case "pgup":
		a.sel = max(0, a.sel-max(1, a.h-8))
	case "pgdn":
		a.sel = min(max(0, n-1), a.sel+max(1, a.h-8))
	case "home", "g":
		a.sel = 0
	case "end", "G":
		a.sel = max(0, n-1)
	case "left", "right", "h", "l":
		back := k == "left" || k == "h"
		switch page {
		case "top":
			d := 1
			if back {
				d = len(dims) - 1
			}
			a.dim = (a.dim + d) % len(dims)
			a.changed()
		case "search":
			if back {
				a.offset = max(0, a.offset-a.pageSize())
			} else if n >= a.pageSize() {
				a.offset += a.pageSize()
			}
			a.changed()
		}
	case "tab", "backtab":
		d := 1
		if k == "backtab" {
			d = len(pageKeys) - 1
		}
		a.setPage((a.pg + d) % len(pageKeys))
	case "1", "2", "3":
		a.setPage(int(k[0] - '1'))
	case "t":
		a.rng = (a.rng + 1) % len(ranges)
		a.offset = 0
		a.changed()
	case "T":
		a.rng = (a.rng + len(ranges) - 1) % len(ranges)
		a.offset = 0
		a.changed()
	case "u":
		a.unit = (a.unit + 1) % len(units)
		a.kick()
	case "p", " ":
		a.paused = !a.paused
		if !a.paused {
			a.kick()
		}
	case "+", "=":
		a.ival = min(len(intervals)-1, a.ival+1)
		a.kick()
	case "-", "_":
		a.ival = max(0, a.ival-1)
		a.kick()
	case "c":
		a.filters = nil
		a.offset = 0
		a.changed()
	case "backspace", "delete":
		if len(a.filters) > 0 {
			a.filters = a.filters[:len(a.filters)-1]
			a.offset = 0
			a.changed()
		}
	case "r":
		a.kick()
	case "/":
		s := []rune{}
		a.input, a.histPos = &s, len(a.hist)
	case "?":
		a.help = true
	case "w":
		a.openBrowser()
	case "enter", "f", "x", "y":
		tg := a.selTarget()
		if tg == nil {
			if page == "search" && k == "enter" {
				s := []rune{}
				a.input, a.histPos = &s, len(a.hist)
			}
			return
		}
		switch {
		case k == "enter" && page == "ifaces":
			if a.ifc == tg.value {
				a.ifc = ""
			} else {
				a.ifc = tg.value
			}
			a.kick()
		case k == "enter":
			a.menu, a.menuSel = tg, 0
		case k == "f":
			a.menu = tg
			a.act("only")
		case k == "x":
			a.menu = tg
			a.act("not")
		case k == "y":
			a.menu = tg
			a.act("copy")
		}
	}
	a.rememberSel()
}

func (a *app) inputKey(k string) {
	switch k {
	case "esc":
		a.input = nil
	case "enter":
		v := strings.TrimSpace(string(*a.input))
		a.input = nil
		if v == "" {
			return
		}
		fs, bad := parseFilters(v)
		if bad != "" {
			a.flash(a.t("tui.bad_filter", "v", bad))
		}
		if len(fs) == 0 {
			return
		}
		if len(a.hist) == 0 || a.hist[len(a.hist)-1] != v {
			a.hist = append(a.hist, v)
			if len(a.hist) > 50 {
				a.hist = a.hist[1:]
			}
		}
		a.filters = append(a.filters, fs...)
		a.offset = 0
		a.changed()
	case "up":
		if a.histPos > 0 {
			a.histPos--
			s := []rune(a.hist[a.histPos])
			a.input = &s
		}
	case "down":
		if a.histPos < len(a.hist)-1 {
			a.histPos++
			s := []rune(a.hist[a.histPos])
			a.input = &s
		} else {
			a.histPos = len(a.hist)
			s := []rune{}
			a.input = &s
		}
	case "backspace":
		if n := len(*a.input); n > 0 {
			*a.input = (*a.input)[:n-1]
		}
	default:
		if utf8.RuneCountInString(k) == 1 {
			*a.input = append(*a.input, []rune(k)...)
		}
	}
}

func (a *app) setPage(p int) {
	if p == a.pg {
		return
	}
	a.pg = p
	a.data = page{}
	a.selKey = ""
	a.changed()
}

func (a *app) resetSel() { a.sel, a.scroll, a.selKey = 0, 0, "" }

// pageSize is the number of records the search page asks for.
func (a *app) pageSize() int { return max(10, a.h-7) }

func (a *app) act(what string) {
	tg := a.menu
	a.menu = nil
	if tg == nil || tg.field == "" {
		return
	}
	switch what {
	case "copy":
		// OSC 52: most modern terminals copy this to the clipboard
		os.Stdout.WriteString("\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(tg.value)) + "\x07")
		a.flash(a.t("pop.copy") + ": " + tg.value)
		return
	case "records":
		keep := a.filters[:0]
		for _, f := range a.filters {
			if f.F != tg.field {
				keep = append(keep, f)
			}
		}
		a.filters = append(keep, filter{F: tg.field, V: tg.value, lab: tg.label})
		a.pg, a.offset, a.data = 2, 0, page{}
	default:
		a.filters = append(a.filters, filter{F: tg.field, V: tg.value, Neg: what == "not", lab: tg.label})
	}
	a.changed()
}

func (a *app) flash(s string) {
	a.msg, a.msgUntil = s, time.Now().Add(3*time.Second)
}

// filterPrefixes maps the typed prefixes to filter fields.
var filterPrefixes = map[string]string{
	"ip": "ip", "host": "ip", "net": "ip", "asn": "asn", "as": "asn", "port": "port",
	"cc": "country", "country": "country", "dev": "exporter", "device": "exporter",
	"app": "app", "client": "client", "server": "server", "proto": "proto", "vlan": "vlan",
}

// parseFilters reads space separated filters like "ip:10.0.0.5 !port:443";
// a term without a prefix is guessed from its form. bad is the first term
// that could not be read.
func parseFilters(s string) (out []filter, bad string) {
	for _, term := range strings.Fields(s) {
		neg := false
		if strings.HasPrefix(term, "!") || (strings.HasPrefix(term, "-") && len(term) > 1) {
			neg, term = true, term[1:]
		}
		var f filter
		if p, v, ok := strings.Cut(term, ":"); ok && filterPrefixes[strings.ToLower(p)] != "" {
			if v == "" {
				if bad == "" {
					bad = term
				}
				continue
			}
			f = filter{F: filterPrefixes[strings.ToLower(p)], V: v}
			switch f.F {
			case "country":
				f.V = strings.ToUpper(v)
			case "asn":
				f.V = strings.TrimPrefix(strings.ToUpper(v), "AS")
			}
		} else {
			f = guessFilter(term)
		}
		f.Neg = neg
		out = append(out, f)
	}
	return out, bad
}

func guessFilter(v string) filter {
	switch {
	case strings.ContainsAny(v, ".:") && strings.Trim(v, "0123456789abcdefABCDEF.:/") == "":
		return filter{F: "ip", V: v}
	case len(v) > 2 && strings.EqualFold(v[:2], "as") && strings.Trim(v[2:], "0123456789") == "":
		return filter{F: "asn", V: v[2:]}
	case strings.Trim(strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(v), "/tcp"), "/udp"), "0123456789") == "":
		return filter{F: "port", V: v}
	case len(v) == 2:
		return filter{F: "country", V: strings.ToUpper(v)}
	}
	return filter{F: "app", V: v}
}

// webURL is the web UI address of the view the TUI shows.
func (a *app) webURL() string {
	var parts []string
	for _, f := range a.filters {
		p := url.QueryEscape(f.F) + ":" + url.QueryEscape(f.V)
		if f.Neg {
			p = "!" + p
		}
		parts = append(parts, p)
	}
	v := map[string]string{"top": "topn", "ifaces": "ifaces", "search": "records"}[pageKeys[a.pg]]
	r := ranges[a.rng]
	if r == "5m" { // the web UI starts at 15 minutes
		r = "15m"
	}
	h := url.Values{"v": {v}, "r": {r}}
	if pageKeys[a.pg] == "top" {
		h.Set("dim", dims[a.dim])
	}
	if len(parts) > 0 {
		h.Set("f", strings.Join(parts, ","))
	}
	return a.opt.URL + "/#" + h.Encode()
}

func (a *app) openBrowser() {
	u := a.webURL()
	if err := OpenBrowser(u); err != nil {
		a.flash(a.t("tui.open_failed", "url", u))
		return
	}
	a.flash(a.t("tui.opened"))
}

// OpenBrowser opens u in the default web browser.
func OpenBrowser(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	case "darwin":
		cmd = exec.Command("open", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	return cmd.Start()
}

// ------------------------------------------------------------ drawing

func (a *app) rowCount() int {
	n := 0
	for _, s := range a.data.sections {
		n += len(s.rows)
	}
	return n
}

func (a *app) selTarget() *target {
	if r := a.selRow(); r != nil {
		return r.target
	}
	return nil
}

func (a *app) selRow() *row {
	i := a.sel
	for si := range a.data.sections {
		s := &a.data.sections[si]
		if i < len(s.rows) {
			return &s.rows[i]
		}
		i -= len(s.rows)
	}
	return nil
}

func (a *app) draw(out *bufio.Writer) {
	a.mu.Lock()
	lines := a.frame()
	a.mu.Unlock()
	out.WriteString("\x1b[H")
	for i, l := range lines {
		out.WriteString(l)
		if i < len(lines)-1 {
			out.WriteString("\r\n")
		}
	}
	out.Flush()
}

// frame lays out the whole screen, one string per terminal row.
func (a *app) frame() []string {
	w, h := a.w, a.h
	if w < 40 || h < 12 {
		return []string{"traffic66: window too small"}
	}
	var lines []string
	emit := func(l line) { lines = append(lines, l.render(w, "")) }

	// status line: pages, range, units, refresh
	hdr := line{{bold, "traffic"}, {bold + blue, "66"}, {"", " "}}
	for i, k := range pageKeys {
		lab := fmt.Sprintf(" %d %s ", i+1, a.t("tui.page."+k))
		if i == a.pg {
			hdr = append(hdr, seg{rev + bold, lab})
		} else {
			hdr = append(hdr, seg{dim, lab})
		}
	}
	hdr = append(hdr, seg{"", "  " + a.t("range."+ranges[a.rng]) + " · " + a.t("tui.unit."+units[a.unit]) + " · " + a.t("tui.every", "s", fmt.Sprint(intervals[a.ival].Seconds()))})
	if a.paused {
		hdr = append(hdr, seg{bold + yellow, "  ‖ " + a.t("tui.paused")})
	} else if a.loading {
		hdr = append(hdr, seg{dim, "  …"})
	}
	if !a.updated.IsZero() && w >= narrow {
		hdr = append(hdr, seg{dim, "  " + a.t("tui.updated", "t", a.updated.Format("15:04:05"))})
	}
	emit(hdr)
	// persistent filter line
	fl := line{{dim, a.t("tui.filter") + ": "}}
	if len(a.filters) == 0 {
		fl = append(fl, seg{dim, a.t("tui.no_filter")})
	}
	for _, f := range a.filters {
		st, word := cyan, "="
		if f.Neg {
			st, word = red, "≠"
		}
		lab := f.lab
		if lab == "" {
			lab = f.V
		}
		fl = append(fl, seg{st, "[" + a.t("field."+f.F) + word + lab + "]"}, seg{"", " "})
	}
	if len(a.filters) > 0 {
		fl = append(fl, seg{dim, "⌫ c"})
	}
	emit(fl)

	// body
	body := h - len(lines) - 1
	var bl []string
	if a.data.err != "" {
		bl = append(bl, line{{red, a.data.err}}.render(w, ""))
	}
	for _, l := range a.data.top {
		bl = append(bl, l.render(w, ""))
	}
	type rl struct {
		text string
		idx  int // row index, -1 for headers and details
	}
	var list []rl
	ri := 0
	for _, s := range a.data.sections {
		cols, keep := visibleCols(s.cols, w)
		widths := layout(cols, w-1)
		if len(s.title) > 0 {
			list = append(list, rl{s.title.render(w, ""), -1})
		}
		hd := line{{"", " "}}
		for i, c := range cols {
			txt := c.head
			if c.right {
				txt = fitRight(txt, widths[i])
			} else {
				txt = fit(txt, widths[i])
			}
			hd = append(hd, seg{dim, txt + " "})
		}
		list = append(list, rl{hd.render(w, ""), -1})
		if len(s.rows) == 0 {
			list = append(list, rl{line{{dim, " " + a.t("tui.nothing")}}.render(w, ""), -1})
		}
		for _, r := range s.rows {
			l := line{{"", " "}}
			if ri == a.sel {
				l = line{{bold, "›"}}
			}
			for i, ci := range keep {
				var c line
				if ci < len(r.cells) {
					c = r.cells[ci]
				}
				cw := c.width()
				if cw > widths[i] {
					st := ""
					if len(c) > 0 {
						st = c[0].style
					}
					c = line{{st, fit(c.plain(), widths[i])}}
				} else if cols[i].right {
					c = append(line{{"", spaces(widths[i] - cw)}}, c...)
				} else {
					c = append(c, seg{"", spaces(widths[i] - cw)})
				}
				l = append(l, c...)
				l = append(l, seg{"", " "})
			}
			base := ""
			if ri == a.sel && a.menu == nil {
				base = rev
			}
			list = append(list, rl{l.render(w, base), ri})
			for _, d := range r.detail {
				list = append(list, rl{d.render(w, ""), -1})
			}
			ri++
		}
		list = append(list, rl{spaces(w), -1})
	}
	var foot []string
	for _, l := range a.data.foot {
		foot = append(foot, l.render(w, ""))
	}
	avail := max(3, body-len(bl)-len(foot))
	// keep the selected row (and what is expanded under it) visible
	selPos, selEnd := 0, 0
	for i, x := range list {
		if x.idx == a.sel {
			selPos, selEnd = i, i
			for j := i + 1; j < len(list) && list[j].idx == -1 && j-i <= avail-2; j++ {
				selEnd = j
			}
			break
		}
	}
	if selPos < a.scroll+1 {
		a.scroll = max(0, selPos-1)
	}
	if selEnd >= a.scroll+avail {
		a.scroll = selEnd - avail + 1
	}
	if a.scroll > len(list)-avail {
		a.scroll = max(0, len(list)-avail)
	}
	menuY := -1
	for i := a.scroll; i < len(list) && i < a.scroll+avail; i++ {
		if list[i].idx == a.sel {
			menuY = len(lines) + len(bl)
		}
		bl = append(bl, list[i].text)
	}
	for len(bl) < body-len(foot) {
		bl = append(bl, spaces(w))
	}
	bl = append(bl, foot...)
	lines = append(lines, bl[:min(len(bl), body)]...)

	// footer
	switch {
	case a.input != nil:
		lines = append(lines, line{{bold, a.t("tui.filter") + ": "}, {"", string(*a.input) + "▏"}, {dim, "  " + a.t("tui.filter_hint")}}.render(w, ""))
	case a.msg != "" && time.Now().Before(a.msgUntil):
		lines = append(lines, line{{green, a.msg}}.render(w, ""))
	default:
		keys := a.t("tui.keys")
		if strWidth(keys) > w {
			keys = "? " + a.t("tui.help_title") + "   q " + a.t("tui.h.quit")
		}
		lines = append(lines, line{{dim, keys}}.render(w, ""))
	}

	// action menu, next to the selected row
	if a.menu != nil && menuY >= 0 {
		items := []string{a.t("pop.only") + "  f", a.t("pop.not") + "  x", a.t("pop.records") + "  r", a.t("pop.copy") + "  y"}
		for i := range items {
			if i == a.menuSel {
				items[i] = "› " + items[i]
			} else {
				items[i] = "  " + items[i]
			}
		}
		box := []line{{{bold, a.menu.label}}}
		for i, it := range items {
			st := ""
			if i == a.menuSel {
				st = rev
			}
			box = append(box, line{{st, it}})
		}
		box = append(box, line{{dim, a.t("tui.close")}})
		mw := 0
		for _, b := range box {
			mw = max(mw, b.width()+2)
		}
		mw = min(mw, w-8)
		left := min(6, w-mw-2)
		top := menuY + 1
		if top+len(box)+2 > len(lines)-1 {
			top = max(2, menuY-len(box)-2)
		}
		lines = overlay(lines, box, top, left, mw)
	}
	if a.help {
		lines = overlay(lines, a.helpBox(), 3, 2, min(w-4, 80))
	}
	return lines
}

// overlay draws a framed box over the lines at row top, column left.
func overlay(lines []string, box []line, top, left, mw int) []string {
	put := func(y int, s string) {
		if y >= 0 && y < len(lines) {
			lines[y] = s
		}
	}
	edge := strings.Repeat("─", mw)
	put(top, spaces(left)+"┌"+edge+"┐")
	for i, b := range box {
		put(top+1+i, spaces(left)+"│"+b.render(mw, "")+"│")
	}
	put(top+1+len(box), spaces(left)+"└"+edge+"┘")
	return lines
}

func (a *app) helpBox() []line {
	box := []line{{{bold, a.t("tui.help_title")}}}
	for _, k := range []struct{ keys, text string }{
		{"1 2 3  Tab", "tui.h.pages"}, {"↑↓ PgUp PgDn", "tui.h.move"}, {"← →", "tui.h.dim"},
		{"Enter", "tui.h.enter"}, {"f  x  y", "tui.h.fx"}, {"/", "tui.h.filter"},
		{"⌫  c", "tui.h.clear"}, {"t  T", "tui.h.range"}, {"u", "tui.h.units"},
		{"+  -", "tui.h.refresh"}, {"p", "tui.h.pause"}, {"w", "tui.h.web"}, {"q", "tui.h.quit"},
	} {
		box = append(box, line{{bold, fit(k.keys, 14)}, {"", " " + a.t(k.text)}})
	}
	box = append(box, line{}, line{{dim, a.t("tui.close")}})
	return box
}

// visibleCols drops the optional columns on narrow terminals, so that text
// is cut only when nothing else can go. keep maps back to the cell index.
func visibleCols(cols []column, w int) ([]column, []int) {
	var out []column
	var keep []int
	for i, c := range cols {
		if c.optional && w < narrow {
			continue
		}
		out = append(out, c)
		keep = append(keep, i)
	}
	return out, keep
}

// layout gives each column a width; flexible columns share what is left.
func layout(cols []column, total int) []int {
	ws := make([]int, len(cols))
	fixed, flex := 0, 0
	for i, c := range cols {
		ws[i] = c.width
		if c.width == 0 {
			flex++
		}
		fixed += c.width + 1
	}
	rest := total - fixed
	for i, c := range cols {
		if c.width == 0 {
			ws[i] = max(6, rest/max(1, flex)-1)
		}
	}
	return ws
}

// ------------------------------------------------------------ formatting

func fmtBps(v float64) string {
	for _, u := range []struct {
		n string
		m float64
	}{{"Gbps", 1e9}, {"Mbps", 1e6}, {"Kbps", 1e3}} {
		if math.Abs(v) >= u.m {
			x := v / u.m
			if math.Abs(x) < 10 {
				return fmt.Sprintf("%.2f %s", x, u.n)
			}
			return fmt.Sprintf("%.1f %s", x, u.n)
		}
	}
	return fmt.Sprintf("%.0f bps", v)
}

func fmtBytes(v float64) string {
	for _, u := range []struct {
		n string
		m float64
	}{{"TB", 1e12}, {"GB", 1e9}, {"MB", 1e6}, {"KB", 1e3}} {
		if v >= u.m {
			x := v / u.m
			if x < 10 {
				return fmt.Sprintf("%.2f %s", x, u.n)
			}
			return fmt.Sprintf("%.1f %s", x, u.n)
		}
	}
	return fmt.Sprintf("%.0f B", v)
}

func fmtPps(v float64) string {
	for _, u := range []struct {
		n string
		m float64
	}{{"Gpps", 1e9}, {"Mpps", 1e6}, {"Kpps", 1e3}} {
		if v >= u.m {
			return fmt.Sprintf("%.1f %s", v/u.m, u.n)
		}
	}
	if v < 10 {
		return fmt.Sprintf("%.1f pps", v)
	}
	return fmt.Sprintf("%.0f pps", v)
}

func pct(v float64) string {
	s := fmt.Sprintf("%.1f%%", v*100)
	if v > 0 {
		s = "+" + s
	}
	return s
}

// delta formats the change from base to cur. It is empty when the base is
// too small to compare against (tiny is the floor), and capped, so that a
// host that was nearly idle reads "new" or ">10×" rather than "+646539%".
func delta(cur, base, tiny float64) string {
	switch {
	case base <= 0 && cur <= tiny:
		return ""
	case base <= tiny:
		if cur > tiny {
			return "new"
		}
		return ""
	case cur/base > 10:
		return ">10×"
	}
	d := (cur - base) / base
	if math.Abs(d) < 0.005 {
		return "±0%"
	}
	if d > 0 {
		return fmt.Sprintf("+%.0f%%", d*100)
	}
	return fmt.Sprintf("%.0f%%", d*100)
}

func barCells(v, maxv float64, width int) string {
	if maxv <= 0 || width <= 0 {
		return ""
	}
	eighths := int(math.Round(v / maxv * float64(width) * 8))
	full := min(width, eighths/8)
	s := strings.Repeat("█", full)
	if r := eighths % 8; r > 0 && full < width {
		s += string([]rune("▏▎▍▌▋▊▉")[r-1])
	}
	return s
}
