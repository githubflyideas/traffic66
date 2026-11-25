package tui

import (
	"strings"
	"testing"
)

func TestParseFilters(t *testing.T) {
	fs, bad := parseFilters("ip:10.0.0.5 net:10.0.0.0/8 asn:AS13335 !port:443 cc:us dev:192.0.2.1 app:DNS 2001:db8::1 ip:")
	want := []filter{{F: "ip", V: "10.0.0.5"}, {F: "ip", V: "10.0.0.0/8"}, {F: "asn", V: "13335"}, {F: "port", V: "443", Neg: true},
		{F: "country", V: "US"}, {F: "exporter", V: "192.0.2.1"}, {F: "app", V: "DNS"}, {F: "ip", V: "2001:db8::1"}}
	if len(fs) != len(want) {
		t.Fatalf("got %+v", fs)
	}
	for i := range want {
		if fs[i] != want[i] {
			t.Errorf("%d: got %+v, want %+v", i, fs[i], want[i])
		}
	}
	if bad != "ip:" {
		t.Errorf("bad = %q", bad)
	}
	// without a prefix the old guessing still applies
	if fs, _ := parseFilters("AS64500 53/udp jp"); fs[0].F != "asn" || fs[1].F != "port" || fs[2].V != "JP" {
		t.Errorf("guess: %+v", fs)
	}
}

func TestDeltaIsCapped(t *testing.T) {
	for _, c := range []struct {
		cur, base, tiny float64
		want            string
	}{
		{100, 50, 1, "+100%"}, {50, 100, 1, "-50%"}, {100, 100, 1, "±0%"},
		{6465390, 1000, 1, ">10×"}, // was shown as +646539%
		{100, 0.5, 1, "new"}, {0.5, 0, 1, ""}, {0, 0, 1, ""},
	} {
		if got := delta(c.cur, c.base, c.tiny); got != c.want {
			t.Errorf("delta(%v, %v) = %q, want %q", c.cur, c.base, got, c.want)
		}
	}
}

func TestNarrowDropsOptionalColumns(t *testing.T) {
	cols := []column{{head: "a"}, {head: "bar", width: 20, optional: true}, {head: "b", width: 5}}
	if c, keep := visibleCols(cols, 70); len(c) != 2 || keep[1] != 2 {
		t.Errorf("70 cols: %v %v", c, keep)
	}
	if c, _ := visibleCols(cols, 120); len(c) != 3 {
		t.Errorf("120 cols: %v", c)
	}
}

func TestFrameFitsAndHasNoWebHints(t *testing.T) {
	noColor = true
	defer func() { noColor = false }()
	a := newApp(Options{URL: "http://x"})
	a.data = page{sections: []section{{cols: []column{{head: "Host"}, {head: "bits/s", width: 11, right: true}, {head: "", width: 20, optional: true}},
		rows: []row{{key: "1", cells: []line{txt(strings.Repeat("very-long-host-name.", 6)), txt("1.00 Mbps"), styled(blue, "████")}, target: &target{"ip", "10.0.0.1", "x"}}}}}}
	for _, w := range []int{70, 120} {
		a.w, a.h = w, 24
		lines := a.frame()
		if len(lines) != 24 {
			t.Fatalf("%d: %d lines", w, len(lines))
		}
		all := strings.Join(lines, "\n")
		if strings.Contains(all, "\x1b[3") {
			t.Errorf("%d: colour with NO_COLOR", w)
		}
		if strings.Contains(strings.ToLower(all), "click") {
			t.Errorf("%d: web-only hint in the TUI", w)
		}
		if !strings.Contains(all, "1.00 Mbps") {
			t.Errorf("%d: the rate was cut before the name", w)
		}
		if w == 70 && strings.Contains(all, "████") {
			t.Errorf("bar kept on a narrow terminal")
		}
	}
}

func TestWebURLMatchesView(t *testing.T) {
	a := newApp(Options{URL: "http://h:8066"})
	a.rng = 0 // 5 min, which the web UI does not have
	a.filters = []filter{{F: "ip", V: "10.0.0.1", Neg: true}}
	u := a.webURL()
	for _, s := range []string{"v=topn", "dim=client", "r=15m", "f=%21ip%3A10.0.0.1"} {
		if !strings.Contains(u, s) {
			t.Errorf("%s lacks %s", u, s)
		}
	}
	a.pg = 1
	if u := a.webURL(); !strings.Contains(u, "v=ifaces") || strings.Contains(u, "dim=") {
		t.Errorf("ifaces: %s", u)
	}
}
