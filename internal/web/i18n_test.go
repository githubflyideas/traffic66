package web

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Every language must have every key, with the same placeholders.
func TestTranslationsComplete(t *testing.T) {
	read := func(name string) map[string]string {
		b, err := fs.ReadFile(FS(), "i18n/"+name)
		if err != nil {
			t.Fatal(err)
		}
		m := map[string]string{}
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return m
	}
	en := read("en.json")
	ph := regexp.MustCompile(`\{\w+\}`)
	files, _ := fs.Glob(FS(), "i18n/*.json")
	// The languages of the UI, by total number of speakers (Ethnologue).
	for _, want := range []string{"en", "zh", "hi", "es", "ar", "fr", "bn", "pt", "ru", "id", "ur", "de", "ja", "vi", "ko", "zh-tw"} {
		if _, err := fs.Stat(FS(), "i18n/"+want+".json"); err != nil {
			t.Errorf("missing language %s", want)
		}
	}
	for _, f := range files {
		m := read(f[len("i18n/"):])
		for k, v := range en {
			tv, ok := m[k]
			if !ok {
				t.Errorf("%s: missing %q", f, k)
				continue
			}
			a, b := ph.FindAllString(v, -1), ph.FindAllString(tv, -1)
			sort.Strings(a)
			sort.Strings(b)
			if strings.Join(a, ",") != strings.Join(b, ",") {
				t.Errorf("%s: %q placeholders %v vs %v", f, k, a, b)
			}
		}
		for k := range m {
			if _, ok := en[k]; !ok {
				t.Errorf("%s: unknown key %q", f, k)
			}
		}
	}
}

// Every key the pages ask for by name is in the English file, and the
// Chinese and Japanese say 国家/地区 and 国・地域.
func TestTranslationsUsed(t *testing.T) {
	b, _ := fs.ReadFile(FS(), "i18n/en.json")
	en := map[string]string{}
	if err := json.Unmarshal(b, &en); err != nil {
		t.Fatal(err)
	}
	used := regexp.MustCompile(`\bt\('([a-z0-9_]+\.[a-z0-9_.]*[a-z0-9])'\s*[,)]`)
	for _, f := range []string{"app.js", "embed.js"} {
		js, err := fs.ReadFile(FS(), f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range used.FindAllStringSubmatch(string(js), -1) {
			if _, ok := en[m[1]]; !ok {
				t.Errorf("%s uses %q, which en.json lacks", f, m[1])
			}
		}
	}
	read := func(l string) map[string]string {
		b, _ := fs.ReadFile(FS(), "i18n/"+l+".json")
		m := map[string]string{}
		json.Unmarshal(b, &m)
		return m
	}
	for k, v := range read("zh") {
		if strings.Count(v, "国家") != strings.Count(v, "国家/地区") {
			t.Errorf("zh %s: 国家 without /地区: %s", k, v)
		}
	}
	for k, v := range read("ja") {
		if strings.Contains(v, "国家") || strings.Contains(v, "国/地域") {
			t.Errorf("ja %s: write 国・地域: %s", k, v)
		}
	}
}

// TestLanguageOrder keeps Traditional Chinese immediately after Korean
// in every UI language list.
func TestLanguageOrder(t *testing.T) {
	check := func(name, src string) {
		t.Helper()
		i := strings.Index(src, "const LANGS =")
		if i < 0 {
			t.Fatalf("%s: language list not found", name)
		}
		j := strings.Index(src[i:], "];")
		if j < 0 {
			t.Fatalf("%s: language list not closed", name)
		}
		block := src[i : i+j]
		list := regexp.MustCompile(`['"]([a-z]{2}(?:-[a-z]+)?)['"]`).FindAllStringSubmatch(block, -1)
		var codes []string
		for _, x := range list {
			codes = append(codes, x[1])
		}
		ko, tw := -1, -1
		for k, c := range codes {
			if c == "ko" {
				ko = k
			}
			if c == "zh-tw" {
				tw = k
			}
		}
		if ko < 0 || tw != ko+1 {
			t.Errorf("%s: ko at %d, zh-tw at %d in %v; want zh-tw immediately after ko", name, ko, tw, codes)
		}
	}
	app, err := fs.ReadFile(FS(), "app.js")
	if err != nil {
		t.Fatal(err)
	}
	check("app.js", string(app))
	emb, err := fs.ReadFile(FS(), "embed.js")
	if err != nil {
		t.Fatal(err)
	}
	check("embed.js", string(emb))
}
