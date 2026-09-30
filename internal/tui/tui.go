// Package tui is the terminal interface. It talks to a running traffic66
// over its HTTP API and needs no third-party terminal library.
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
	"sort"
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
		for _, l := range []string{"en", "zh", "hi", "es", "ar", "fr", "bn", "pt", "ru", "id", "ur", "ja"} {
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
		b.WriteString(base + s.style + t + reset)
		used += strWidth(t)
	}
	b.WriteString(base + spaces(w-used) + reset)
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
	detail []line // shown under the list for the selected row
}

type column struct {
	head  string
	width int  // fixed width; 0 = flexible
	right bool // right-aligned
}

type section struct {
	title string
	cols  []column
	rows  []row
}

type page struct {
	top      []line // fixed lines above the lists
	sections []section
	err      string
}

var pageKeys = []string{"overview", "topn", "sankey", "geo", "threats", "records", "ifaces", "sources"}
var ranges = []string{"15m", "1h", "6h", "24h", "7d", "30d"}
var dims = []string{"client", "server", "conv", "app", "port", "country", "asn", "segment", "exporter", "encap", "vlan"}

type app struct {
	opt    Options
	tr, en dict
	cl     *http.Client

	mu       sync.Mutex
	pg       int
	rng      int
	dim      int
	filters  []filter
	sel      int
	scroll   int
	data     page
	loading  bool
	updated  time.Time
	menu     *target
	menuSel  int
	search   *[]rune
	msg      string
	msgUntil time.Time
	names    map[string]string
	ifc      string // exporter|ifindex of the selected interface
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

// ------------------------------------------------------------ run

// Run starts the terminal UI and blocks until the user quits or ctx ends.
func Run(ctx context.Context, opt Options) error {
	if !isTerminal() {
		return errors.New("traffic66 tui needs an interactive terminal")
	}
	if opt.URL == "" {
		opt.URL = "http://127.0.0.1:8066"
	}
	lang := detectLang(opt.Lang)
	tr, en := loadDict(lang)
	a := &app{opt: opt, tr: tr, en: en, cl: &http.Client{Timeout: 20 * time.Second}, rng: 3, names: map[string]string{}, refresh: make(chan struct{}, 1)}
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
		case k := <-keys:
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
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.refresh:
		case <-t.C:
			a.mu.Lock()
			busy := a.menu != nil || a.search != nil
			a.mu.Unlock()
			if busy {
				continue
			}
		}
		a.mu.Lock()
		a.loading = true
		pg, rng, dim, filters, ifc := a.pg, a.rng, a.dim, append([]filter{}, a.filters...), a.ifc
		a.mu.Unlock()
		p := a.load(pg, rng, dim, filters, ifc)
		a.mu.Lock()
		if pg == a.pg && rng == a.rng && dim == a.dim {
			a.data = p
			a.updated = time.Now()
			n := a.rowCount()
			if a.sel >= n {
				a.sel = max(0, n-1)
			}
		}
		a.loading = false
		a.mu.Unlock()
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

func (a *app) query(rng int, filters []filter, extra map[string]string) url.Values {
	q := url.Values{"range": {ranges[rng]}}
	if len(filters) > 0 {
		b, _ := json.Marshal(filters)
		q.Set("f", string(b))
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

func (a *app) key(k string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if k == "ctrl-c" {
		a.quit = true
		return
	}
	if a.search != nil {
		switch k {
		case "esc":
			a.search = nil
		case "enter":
			if v := strings.TrimSpace(string(*a.search)); v != "" {
				f := guessFilter(v)
				a.filters = append(a.filters, f)
				a.resetSel()
				a.kick()
			}
			a.search = nil
		case "backspace":
			if n := len(*a.search); n > 0 {
				*a.search = (*a.search)[:n-1]
			}
		default:
			if utf8.RuneCountInString(k) == 1 {
				*a.search = append(*a.search, []rune(k)...)
			}
		}
		return
	}
	if a.menu != nil {
		acts := []string{"only", "not", "records", "copy"}
		switch k {
		case "esc", "q":
			a.menu = nil
		case "up":
			a.menuSel = (a.menuSel + len(acts) - 1) % len(acts)
		case "down", "tab":
			a.menuSel = (a.menuSel + 1) % len(acts)
		case "f":
			a.act("only")
		case "x":
			a.act("not")
		case "y":
			a.act("copy")
		case "enter":
			a.act(acts[a.menuSel])
		}
		return
	}
	n := a.rowCount()
	switch k {
	case "q":
		a.quit = true
	case "up", "k":
		if a.sel > 0 {
			a.sel--
		}
	case "down", "j":
		if a.sel < n-1 {
			a.sel++
		}
	case "pgup":
		a.sel = max(0, a.sel-(a.h-8))
	case "pgdn":
		a.sel = min(max(0, n-1), a.sel+(a.h-8))
	case "home", "g":
		a.sel = 0
	case "end", "G":
		a.sel = max(0, n-1)
	case "left", "right":
		if a.pg == 1 {
			d := 1
			if k == "left" {
				d = len(dims) - 1
			}
			a.dim = (a.dim + d) % len(dims)
			a.resetSel()
			a.kick()
		}
	case "tab", "backtab":
		d := 1
		if k == "backtab" {
			d = len(pageKeys) - 1
		}
		a.pg = (a.pg + d) % len(pageKeys)
		a.resetSel()
		a.kick()
	case "t":
		a.rng = (a.rng + 1) % len(ranges)
		a.kick()
	case "T":
		a.rng = (a.rng + len(ranges) - 1) % len(ranges)
		a.kick()
	case "c":
		a.filters = nil
		a.resetSel()
		a.kick()
	case "backspace":
		if len(a.filters) > 0 {
			a.filters = a.filters[:len(a.filters)-1]
			a.resetSel()
			a.kick()
		}
	case "r":
		a.kick()
	case "/":
		s := []rune{}
		a.search = &s
	case "w":
		a.openBrowser()
	case "enter", "f", "x":
		if tg := a.selTarget(); tg != nil {
			if k == "enter" {
				if a.pg == 6 { // interface list: enter selects
					a.ifc = tg.value
					a.kick()
					return
				}
				a.menu, a.menuSel = tg, 0
				return
			}
			a.menu = tg
			if k == "f" {
				a.act("only")
			} else {
				a.act("not")
			}
		}
	default:
		if len(k) == 1 && k[0] >= '1' && k[0] <= '8' {
			a.pg = int(k[0] - '1')
			a.resetSel()
			a.kick()
		}
	}
}

func (a *app) resetSel() { a.sel, a.scroll = 0, 0 }

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
		a.flash(tg.value)
		return
	case "records":
		keep := a.filters[:0]
		for _, f := range a.filters {
			if f.F != tg.field {
				keep = append(keep, f)
			}
		}
		a.filters = append(keep, filter{F: tg.field, V: tg.value, lab: tg.label})
		a.pg = 5
	default:
		a.filters = append(a.filters, filter{F: tg.field, V: tg.value, Neg: what == "not", lab: tg.label})
	}
	a.resetSel()
	a.kick()
}

func (a *app) flash(s string) {
	a.msg, a.msgUntil = s, time.Now().Add(3*time.Second)
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

func (a *app) openBrowser() {
	var parts []string
	for _, f := range a.filters {
		p := url.QueryEscape(f.F) + ":" + url.QueryEscape(f.V)
		if f.Neg {
			p = "!" + p
		}
		parts = append(parts, p)
	}
	h := url.Values{"v": {pageKeys[a.pg]}, "r": {ranges[a.rng]}}
	if a.pg == 1 {
		h.Set("dim", dims[a.dim])
	}
	if len(parts) > 0 {
		h.Set("f", strings.Join(parts, ","))
	}
	u := a.opt.URL + "/#" + h.Encode()
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	case "darwin":
		cmd = exec.Command("open", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	if err := cmd.Start(); err != nil {
		a.flash(a.t("tui.open_failed", "url", u))
		return
	}
	a.flash(a.t("tui.opened"))
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
	i := a.sel
	for _, s := range a.data.sections {
		if i < len(s.rows) {
			return s.rows[i].target
		}
		i -= len(s.rows)
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
	defer a.mu.Unlock()
	w, h := a.w, a.h
	if w < 40 || h < 12 {
		out.WriteString("\x1b[H\x1b[2J" + "traffic66: window too small")
		out.Flush()
		return
	}
	var lines []string
	emit := func(l line, base string) { lines = append(lines, l.render(w, base)) }

	// header
	hdr := line{{bold, "traffic"}, {bold + blue, "66"}, {"", "  "}}
	hdr = append(hdr, seg{dim, a.t("nav." + pageKeys[a.pg])}, seg{"", "  "}, seg{"", "[" + a.t("range."+ranges[a.rng]) + "]"})
	if a.loading {
		hdr = append(hdr, seg{dim, "  …"})
	}
	if !a.updated.IsZero() {
		hdr = append(hdr, seg{dim, "  " + a.t("tui.updated", "t", a.updated.Format("15:04:05"))})
	}
	emit(hdr, "")
	// tabs
	tabs := line{}
	for i, k := range pageKeys {
		label := fmt.Sprintf(" %d %s ", i+1, a.t("nav."+k))
		if i == a.pg {
			tabs = append(tabs, seg{rev + bold, label})
		} else {
			tabs = append(tabs, seg{"", label})
		}
	}
	emit(tabs, "")
	// filters
	fl := line{}
	if len(a.filters) == 0 {
		fl = append(fl, seg{dim, a.t("filters.hint")})
	}
	for _, f := range a.filters {
		st, word := cyan, a.t("filters.only")
		if f.Neg {
			st, word = red, a.t("filters.not")
		}
		lab := f.lab
		if lab == "" {
			lab = f.V
		}
		fl = append(fl, seg{st, "[" + word + " " + a.t("field."+f.F) + ": " + lab + "]"}, seg{"", " "})
	}
	emit(fl, "")

	// body
	body := h - len(lines) - 1
	var bl []string
	bemit := func(l line, base string) { bl = append(bl, l.render(w, base)) }
	if a.data.err != "" {
		bemit(line{{red, a.data.err}}, "")
	}
	for _, l := range a.data.top {
		bemit(l, "")
	}
	// list area with selection and scrolling
	type rl struct {
		text string
		idx  int // row index, -1 for headers
	}
	var list []rl
	ri := 0
	for _, s := range a.data.sections {
		widths := layout(s.cols, w-1)
		if s.title != "" {
			list = append(list, rl{line{{bold, s.title}}.render(w, ""), -1})
		}
		hd := line{{"", " "}}
		for i, c := range s.cols {
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
			for i := range s.cols {
				var c line
				if i < len(r.cells) {
					c = r.cells[i]
				}
				cw := c.width()
				if cw > widths[i] {
					// cut the cell
					txt := ""
					for _, sg := range c {
						txt += sg.text
					}
					st := ""
					if len(c) > 0 {
						st = c[0].style
					}
					c = line{{st, fit(txt, widths[i])}}
				} else if s.cols[i].right {
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
			ri++
		}
		list = append(list, rl{spaces(w), -1})
	}
	// detail of the selected row
	var detail []string
	if r := a.selRow(); r != nil {
		for _, d := range r.detail {
			detail = append(detail, d.render(w, ""))
		}
	}
	avail := body - len(bl) - len(detail)
	if avail < 3 {
		avail = 3
	}
	// keep the selected row visible
	selPos := 0
	for i, x := range list {
		if x.idx == a.sel {
			selPos = i
			break
		}
	}
	if selPos < a.scroll+1 {
		a.scroll = max(0, selPos-1)
	}
	if selPos >= a.scroll+avail {
		a.scroll = selPos - avail + 1
	}
	if a.scroll > len(list)-avail {
		a.scroll = max(0, len(list)-avail)
	}
	for i := a.scroll; i < len(list) && i < a.scroll+avail; i++ {
		bl = append(bl, list[i].text)
	}
	bl = append(bl, detail...)
	for len(bl) < body {
		bl = append(bl, spaces(w))
	}
	lines = append(lines, bl[:body]...)

	// footer
	if a.search != nil {
		lines = append(lines, line{{bold, a.t("tui.search") + ": "}, {"", string(*a.search) + "▏"}}.render(w, ""))
	} else if a.msg != "" && time.Now().Before(a.msgUntil) {
		lines = append(lines, line{{green, a.msg}}.render(w, ""))
	} else {
		lines = append(lines, line{{dim, a.t("tui.help")}}.render(w, ""))
	}

	// action menu overlay
	if a.menu != nil {
		items := []string{a.t("pop.only") + "   f", a.t("pop.not") + "   x", a.t("pop.records") + "   ↵", a.t("pop.copy") + "   y"}
		mw := strWidth(a.menu.label) + 4
		for _, it := range items {
			mw = max(mw, strWidth(it)+4)
		}
		mw = min(mw, w-4)
		top := 4
		left := 3
		box := []line{{{bold, a.menu.label}}}
		for i, it := range items {
			st := ""
			if i == a.menuSel {
				st = rev
			}
			box = append(box, line{{st, it}})
		}
		box = append(box, line{{dim, a.t("tui.close")}})
		for i, b := range box {
			y := top + i
			if y >= len(lines) {
				break
			}
			lines[y] = spaces(left) + "│" + b.render(mw, "") + "│"
		}
	}

	out.WriteString("\x1b[H")
	for i, l := range lines {
		out.WriteString(l)
		if i < len(lines)-1 {
			out.WriteString("\r\n")
		}
	}
	out.Flush()
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

func pct(v float64) string {
	s := fmt.Sprintf("%.1f%%", v*100)
	if v > 0 {
		s = "+" + s
	}
	return s
}

func barCells(v, maxv float64, width int) string {
	if maxv <= 0 || width <= 0 {
		return ""
	}
	eighths := int(math.Round(v / maxv * float64(width) * 8))
	full := eighths / 8
	s := strings.Repeat("█", full)
	if r := eighths % 8; r > 0 && full < width {
		s += string([]rune("▏▎▍▌▋▊▉")[r-1])
	}
	return s
}

// sortedKeys is used for deterministic output of maps.
func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
