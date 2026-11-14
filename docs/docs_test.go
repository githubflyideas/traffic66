package docs

import (
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"
)

var (
	headingRe = regexp.MustCompile(`^(#{1,4}) (.*)$`)
	anchorRe  = regexp.MustCompile(`<a id="([\w-]+)"></a>`)
	linkRe    = regexp.MustCompile(`\]\(#([^)]+)\)`)
	slugDrop  = regexp.MustCompile(`[^\p{L}\p{N}\s_-]`)
)

// outline is a manual's heading levels (outside code blocks), its anchors
// (headings and <a id>) and the in-page links it makes.
func outline(s string) (levels string, anchors map[string]bool, links []string) {
	anchors = map[string]bool{}
	code := false
	for _, l := range strings.Split(s, "\n") {
		if strings.HasPrefix(l, "```") {
			code = !code
			continue
		}
		if code {
			continue
		}
		if m := headingRe.FindStringSubmatch(l); m != nil {
			levels += string(rune('0' + len(m[1])))
			anchors[strings.ReplaceAll(strings.TrimSpace(slugDrop.ReplaceAllString(strings.ToLower(m[2]), "")), " ", "-")] = true
		}
		for _, m := range anchorRe.FindAllStringSubmatch(l, -1) {
			anchors[m[1]] = true
		}
		for _, m := range linkRe.FindAllStringSubmatch(l, -1) {
			links = append(links, m[1])
		}
	}
	return
}

// Every translation of the manual has the sections of the English one, and
// every link inside a manual leads somewhere in it.
func TestManualsMatch(t *testing.T) {
	read := func(n string) string {
		b, err := fs.ReadFile(FS, n)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	en, _, _ := outline(read("MANUAL.md"))
	files, _ := fs.Glob(FS, "MANUAL*.md")
	if len(files) != 15 {
		t.Errorf("%d manuals, want 15", len(files))
	}
	for _, f := range files {
		levels, anchors, links := outline(read(f))
		if levels != en {
			t.Errorf("%s: headings %s, English %s", f, levels, en)
		}
		for _, l := range links {
			if !anchors[l] {
				t.Errorf("%s: link #%s leads nowhere", f, l)
			}
		}
	}
}

// Chinese says 国家/地区 and Japanese 国・地域, never the word for country alone.
func TestCountryWording(t *testing.T) {
	zh := regexp.MustCompile(`国家(?:/地区)?`)
	for _, f := range []string{"MANUAL.zh.md", "README.zh.md"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range zh.FindAllString(string(b), -1) {
			if m != "国家/地区" {
				t.Errorf("%s: %q without /地区", f, m)
			}
		}
	}
	for _, f := range []string{"MANUAL.ja.md", "README.ja.md"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if s := string(b); strings.Contains(s, "国家") || strings.Contains(s, "国/地域") || strings.Contains(s, "国／地域") {
			t.Errorf("%s: write 国・地域", f)
		}
	}
}
