package tui

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

func testApp() *app {
	a := newApp(Options{URL: "http://127.0.0.1:8066", Lang: "en"})
	a.data = page{sections: []section{{
		cols: []column{{head: "Host"}, {head: "bits/s", width: 11, right: true}},
		rows: []row{
			{key: "a", cells: []line{txt("10.0.0.1"), txt("1.00 Mbps")}, target: &target{field: "ip", value: "10.0.0.1", label: "10.0.0.1"}},
			{key: "b", cells: []line{txt("10.0.0.2"), txt("2.00 Mbps")}, target: &target{field: "ip", value: "10.0.0.2", label: "10.0.0.2"}},
		},
	}}}
	a.rememberSel()
	return a
}

func TestKeysNavigateAndQuit(t *testing.T) {
	a := testApp()
	if a.rowCount() != 2 || a.selTarget() == nil || a.selTarget().value != "10.0.0.1" {
		t.Fatalf("start: count=%d sel=%v", a.rowCount(), a.selTarget())
	}
	a.key("down")
	if a.sel != 1 || a.selKey != "b" {
		t.Fatalf("down: sel=%d key=%s", a.sel, a.selKey)
	}
	a.key("up")
	if a.sel != 0 {
		t.Fatalf("up: sel=%d", a.sel)
	}
	a.key("end")
	if a.sel != 1 {
		t.Fatalf("end: sel=%d", a.sel)
	}
	a.key("home")
	if a.sel != 0 {
		t.Fatalf("home: sel=%d", a.sel)
	}
	a.key("pgdn")
	a.key("pgup")
	a.key("t")
	if a.rng == 1 {
		t.Fatal("t did not advance the range")
	}
	wantRng := a.rng
	a.key("T")
	if a.rng != (wantRng+len(ranges)-1)%len(ranges) {
		t.Fatalf("T: rng=%d", a.rng)
	}
	a.key("u")
	a.key("+")
	a.key("-")
	a.key("p")
	if !a.paused {
		t.Fatal("p did not pause")
	}
	a.key(" ")
	if a.paused {
		t.Fatal("space did not resume")
	}
	a.key("1")
	if pageKeys[a.pg] != "top" && a.pg != 0 {
		t.Fatalf("page 1: %d", a.pg)
	}
	a.key("tab")
	if a.pg == 0 {
		t.Fatal("tab did not change page")
	}
	a.key("backtab")
	a.key("2")
	a.key("3")
	a.setPage(0)
	a.key("r")
	a.key("c")
	if len(a.filters) != 0 {
		t.Fatalf("c left filters: %+v", a.filters)
	}
	a.key("?")
	if !a.help {
		t.Fatal("? did not open help")
	}
	lines := a.frame()
	all := strings.Join(lines, "\n")
	if !strings.Contains(all, "┌") || len(a.helpBox()) < 5 {
		t.Fatalf("help frame missing box: %s", all[:min(200, len(all))])
	}
	a.key("q") // help swallows the first key
	if a.help || a.quit {
		t.Fatal("a key in help should only close help")
	}
	a.key("ctrl-c")
	if !a.quit {
		t.Fatal("ctrl-c did not quit")
	}
	a.quit = false
	a.key("q")
	if !a.quit {
		t.Fatal("q did not quit")
	}
}

func TestKeysFilterMenuAndInput(t *testing.T) {
	a := testApp()
	a.h = 24
	if a.pageSize() < 10 {
		t.Fatalf("pageSize=%d", a.pageSize())
	}
	a.key("enter")
	if a.menu == nil || a.menu.value != "10.0.0.1" {
		t.Fatalf("enter menu: %+v", a.menu)
	}
	a.key("down")
	a.key("up")
	a.key("esc")
	if a.menu != nil {
		t.Fatal("esc did not close the menu")
	}
	a.key("f")
	if len(a.filters) != 1 || a.filters[0].V != "10.0.0.1" || a.filters[0].Neg {
		t.Fatalf("f filter: %+v", a.filters)
	}
	a.key("backspace")
	if len(a.filters) != 0 {
		t.Fatalf("backspace: %+v", a.filters)
	}
	a.key("x")
	if len(a.filters) != 1 || !a.filters[0].Neg {
		t.Fatalf("x filter: %+v", a.filters)
	}
	a.key("c")
	a.key("enter")
	a.act("records")
	if pageKeys[a.pg] != "search" && a.pg != 2 {
		t.Fatalf("records page=%d", a.pg)
	}
	a.setPage(0)
	a.data = testApp().data
	a.rememberSel()
	a.key("y") // copy via OSC 52
	if a.msg == "" {
		t.Fatal("y/copy left no flash")
	}
	a.flash("hello")
	if a.msg != "hello" || a.msgUntil.IsZero() {
		t.Fatal("flash did not set a message")
	}

	a.key("/")
	if a.input == nil {
		t.Fatal("/ did not start input")
	}
	for _, r := range "ip:10.1.1.1" {
		a.inputKey(string(r))
	}
	if string(*a.input) != "ip:10.1.1.1" {
		t.Fatalf("typed %q", string(*a.input))
	}
	a.inputKey("backspace")
	a.inputKey("1")
	a.inputKey("enter")
	if a.input != nil {
		t.Fatal("enter did not close input")
	}
	if len(a.filters) == 0 || len(a.hist) == 0 {
		t.Fatalf("accepted filter not kept: filters=%v hist=%v", a.filters, a.hist)
	}
	a.key("/")
	a.inputKey("up")
	if a.input == nil || string(*a.input) != a.hist[0] {
		t.Fatalf("hist up: %v", a.input)
	}
	a.inputKey("down")
	a.inputKey("esc")
	if a.input != nil {
		t.Fatal("esc did not cancel input")
	}
	st := a.state()
	if st.pg != a.pg {
		t.Fatalf("state %+v", st)
	}
	a.sel = 99
	a.restoreSel()
	if a.sel >= a.rowCount() {
		t.Fatalf("restoreSel left sel=%d", a.sel)
	}

	// too-small window
	a.w, a.h = 20, 8
	if lines := a.frame(); len(lines) != 1 || !strings.Contains(lines[0], "too small") {
		t.Errorf("small frame: %v", lines)
	}

	var buf bytes.Buffer
	a.w, a.h = 120, 24
	a.help = true
	a.draw(bufio.NewWriter(&buf))
	if !strings.Contains(buf.String(), "traffic") {
		t.Errorf("draw: %q", buf.String()[:min(80, buf.Len())])
	}

	if got := fmtBps(1.5e9); !strings.Contains(got, "Gbps") {
		t.Errorf("fmtBps GB: %s", got)
	}
	if got := fmtBps(15e6); !strings.Contains(got, "Mbps") {
		t.Errorf("fmtBps MB: %s", got)
	}
	if got := fmtBps(12); !strings.Contains(got, "bps") {
		t.Errorf("fmtBps small: %s", got)
	}
	if got := fmtBytes(2e12); !strings.Contains(got, "TB") {
		t.Errorf("fmtBytes TB: %s", got)
	}
	if got := fmtBytes(15e6); !strings.Contains(got, "MB") {
		t.Errorf("fmtBytes MB: %s", got)
	}
	if rw := runeWidth(0); rw != 0 {
		t.Errorf("runeWidth(0)=%d", rw)
	}
	if rw := runeWidth('中'); rw != 2 {
		t.Errorf("runeWidth(中)=%d", rw)
	}
	if rw := runeWidth('\n'); rw != 0 {
		t.Errorf("control width=%d", rw)
	}
	// a Devanagari vowel sign should be zero-width
	if r, _ := utf8.DecodeRuneInString("ि"); isIndicMark(r) {
		if runeWidth(r) != 0 {
			t.Errorf("indic mark width=%d", runeWidth(r))
		}
	}
	if got := fitRight("ab", 5); got != "   ab" {
		t.Errorf("fitRight=%q", got)
	}
}
