package tui

import (
	"strings"
	"testing"
)

func TestLineWidthPlainRender(t *testing.T) {
	l := line{{"", "ab"}, {dim, "cd"}}
	if l.width() != 4 {
		t.Errorf("width = %d", l.width())
	}
	if l.plain() != "abcd" {
		t.Errorf("plain = %q", l.plain())
	}
	// rendering cuts at the width (with an ellipsis) and pads short lines
	out := l.render(3, "")
	vis := strings.NewReplacer(reset, "", dim, "").Replace(out)
	if vis != "ab…" {
		t.Errorf("render(3) visible = %q, want %q", vis, "ab…")
	}
	out = line{{"", "x"}}.render(4, "")
	vis = strings.ReplaceAll(out, reset, "")
	if vis != "x   " {
		t.Errorf("render pad = %q", vis)
	}
}

func TestSpark(t *testing.T) {
	if got := spark(nil, 10, 5); got != "" {
		t.Errorf("empty spark = %q", got)
	}
	if got := spark([]float64{1}, 0, 5); got != "" {
		t.Errorf("zero-width spark = %q", got)
	}
	if got := spark([]float64{5}, 1, 5); got != "█" {
		t.Errorf("full spark = %q", got)
	}
	if got := spark([]float64{1, 2}, 2, 2); got != "▅█" {
		t.Errorf("scaled spark = %q", got)
	}
	// negative samples are gaps, drawn as spaces
	if got := spark([]float64{-1, -2}, 2, 0); got != "  " {
		t.Errorf("gap spark = %q", got)
	}
}

func TestMeasureAndFormats(t *testing.T) {
	bps := measure{unit: 0, secs: 60}
	if got := bps.val(6000, 120); got != 800 {
		t.Errorf("bps val = %v, want wire*8/secs", got)
	}
	if got := bps.fmt(800); got != "800 bps" {
		t.Errorf("bps fmt = %q", got)
	}
	byt := measure{unit: 1, secs: 60}
	if got := byt.val(6000, 120); got != 6000 {
		t.Errorf("bytes val = %v", got)
	}
	if got := byt.fmt(1500); got != "1.50 KB" {
		t.Errorf("bytes fmt = %q", got)
	}
	pps := measure{unit: 2, secs: 60}
	if got := pps.val(6000, 120); got != 2 {
		t.Errorf("pps val = %v", got)
	}
	if got := pps.fmt(2); got != "2.0 pps" {
		t.Errorf("pps fmt = %q", got)
	}
	for _, tc := range []struct {
		got, want string
	}{
		{fmtBytes(999), "999 B"},
		{fmtBytes(1e6), "1.00 MB"},
		{fmtBytes(2.5e9), "2.50 GB"},
		{fmtPps(500), "500 pps"},
		{fmtPps(1500), "1.5 Kpps"},
		{fmtBps(2.5e6), "2.50 Mbps"},
		{pct(0.05), "+5.0%"},
		{pct(-0.05), "-5.0%"},
		{pct(0), "0.0%"},
	} {
		if tc.got != tc.want {
			t.Errorf("got %q, want %q", tc.got, tc.want)
		}
	}
}

func TestBarCells(t *testing.T) {
	for _, tc := range []struct {
		v, maxv float64
		w       int
		want    string
	}{
		{1, 1, 4, "████"},
		{0.5, 1, 4, "██"},
		{0.25, 1, 2, "▌"},
		{1, 0, 4, ""},
		{1, 1, 0, ""},
		{0, 1, 4, ""},
	} {
		if got := barCells(tc.v, tc.maxv, tc.w); got != tc.want {
			t.Errorf("barCells(%v, %v, %d) = %q, want %q", tc.v, tc.maxv, tc.w, got, tc.want)
		}
	}
}

func TestFirstSentence(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"One. Two", "One."},
		{"No stop here", "No stop here"},
		{"一。二", "一。"},
		{". leading", ". leading"}, // a separator at position 0 does not cut
	} {
		if got := firstSentence(tc.in); got != tc.want {
			t.Errorf("firstSentence(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestHostAndCC(t *testing.T) {
	tr, en := loadDict("en")
	a := &app{tr: tr, en: en, names: map[string]string{"10.0.0.1": "fileserver"}}
	if got := a.hostLabel("10.0.0.1"); got != "fileserver 10.0.0.1" {
		t.Errorf("hostLabel = %q", got)
	}
	if got := a.hostLabel("10.0.0.2"); got != "10.0.0.2" {
		t.Errorf("hostLabel unknown = %q", got)
	}
	if got := a.host("10.0.0.1").plain(); got != "fileserver 10.0.0.1" {
		t.Errorf("host line = %q", got)
	}
	if got := a.cc("__internal__"); got != a.t("internal") || got == "" {
		t.Errorf("cc internal = %q", got)
	}
	if got := a.cc(""); got != a.t("unknown") {
		t.Errorf("cc empty = %q", got)
	}
	if got := a.cc("JP"); got != "JP" {
		t.Errorf("cc country = %q", got)
	}
	if got := ipTarget(a, "10.0.0.1"); got.field != "ip" || got.label != "fileserver 10.0.0.1" {
		t.Errorf("ipTarget = %+v", got)
	}
}

func TestQuery(t *testing.T) {
	q := query(2, "", map[string]string{"dim": "dir"})
	if q.Get("range") != "30m" || q.Get("dim") != "dir" || q.Has("f") {
		t.Errorf("query = %v", q)
	}
	q = query(0, "[]", nil)
	if q.Has("f") {
		t.Errorf("empty filter list sent: %v", q)
	}
	q = query(0, `[{"f":"ip","v":"10.0.0.1"}]`, nil)
	if !strings.Contains(q.Get("f"), "10.0.0.1") {
		t.Errorf("filters dropped: %v", q)
	}
	// an extra parameter wins over the defaults
	q = query(0, "", map[string]string{"range": "custom"})
	if q.Get("range") != "custom" {
		t.Errorf("extra did not win: %v", q)
	}
}

func TestGuessFilter(t *testing.T) {
	for _, tc := range []struct {
		in   string
		f, v string
	}{
		{"10.0.0.1", "ip", "10.0.0.1"},
		{"2001:db8::1", "ip", "2001:db8::1"},
		{"as64500", "asn", "64500"},
		{"AS64500", "asn", "64500"},
		{"443/tcp", "port", "443/tcp"},
		{"53", "port", "53"},
		{"JP", "country", "JP"},
		{"jp", "country", "JP"},
		{"https", "app", "https"},
	} {
		got := guessFilter(tc.in)
		if got.F != tc.f || got.V != tc.v {
			t.Errorf("guessFilter(%q) = {%s %s}, want {%s %s}", tc.in, got.F, got.V, tc.f, tc.v)
		}
	}
}

func TestDetectLang(t *testing.T) {
	for _, tc := range []struct{ env, want string }{
		{"ja_JP.UTF-8", "ja"},
		{"de_DE", "de"},
		{"xx_YY", "en"},
		{"", "en"},
		{"zh_TW.UTF-8", "zh-tw"},
		{"zh_HK", "zh-tw"},
		{"zh_CN", "zh"},
	} {
		t.Setenv("LC_ALL", "")
		t.Setenv("LC_MESSAGES", "")
		t.Setenv("LANGUAGE", "")
		t.Setenv("LANG", tc.env)
		if got := detectLang(""); got != tc.want {
			t.Errorf("detectLang with LANG=%q = %q, want %q", tc.env, got, tc.want)
		}
	}
	// an explicit choice beats the environment
	t.Setenv("LANG", "ja_JP")
	if got := detectLang("fr"); got != "fr" {
		t.Errorf("explicit = %q", got)
	}
}

func TestLoadDictTraditional(t *testing.T) {
	tr, en := loadDict("zh-tw")
	if len(tr) == 0 {
		t.Fatal("zh-tw dictionary did not load")
	}
	if len(en) == 0 {
		t.Fatal("en fallback dictionary did not load")
	}
	missing, _ := loadDict("xx")
	if len(missing) != 0 {
		t.Errorf("unknown language loaded %d keys", len(missing))
	}
}
