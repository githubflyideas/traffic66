package tui

import "testing"

func TestFit(t *testing.T) {
	cases := []struct {
		in   string
		w    int
		want string
	}{
		{"abc", 5, "abc  "}, {"abcdef", 4, "abc…"}, {"中文字符", 5, "中文…"}, {"中文", 4, "中文"}, {"日本語", 6, "日本語"}, {"x", 0, ""},
	}
	for _, c := range cases {
		if got := fit(c.in, c.w); got != c.want || strWidth(got) != c.w {
			t.Errorf("fit(%q,%d)=%q want %q", c.in, c.w, got, c.want)
		}
	}
	if fitRight("12", 4) != "  12" {
		t.Error("fitRight")
	}
}
