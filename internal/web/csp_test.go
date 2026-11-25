package web

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// The policy must not break the page it is sent with: every script the
// page loads is a file of its own, and nothing runs inline.
func TestCSPFitsIndex(t *testing.T) {
	for _, want := range []string{"script-src 'self'", "object-src 'none'", "img-src 'self' data:"} {
		if !strings.Contains(CSP, want) {
			t.Errorf("CSP lacks %q: %s", want, CSP)
		}
	}
	for _, bad := range []string{"unsafe-eval", "http:", "https:", "*", "frame-ancestors"} {
		if strings.Contains(CSP, bad) {
			t.Errorf("CSP should not contain %q: %s", bad, CSP)
		}
	}
	b, err := fs.ReadFile(FS(), "index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	for _, m := range regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`).FindAllStringSubmatch(html, -1) {
		if !strings.Contains(m[1], "src=") || strings.TrimSpace(m[2]) != "" {
			t.Errorf("inline script in index.html would be blocked: %.80q", m[0])
		}
		if regexp.MustCompile(`src="(https?:)?//`).MatchString(m[1]) {
			t.Errorf("script from another host would be blocked: %q", m[1])
		}
	}
	if m := regexp.MustCompile(`(?i)\son[a-z]+\s*=`).FindString(html); m != "" {
		t.Errorf("inline event handler in index.html would be blocked: %q", m)
	}
	for _, f := range []string{"app.js", "theme.js", "embed.js"} {
		js, err := fs.ReadFile(FS(), f)
		if err != nil {
			t.Fatal(err)
		}
		if regexp.MustCompile(`\beval\(|new Function\(|\son(click|load|error|change|input|submit)=`).Match(js) {
			t.Errorf("%s uses eval or inline handlers, which the policy blocks", f)
		}
	}
}
