package web

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"sort"
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
	for _, want := range []string{"en", "zh", "hi", "es", "ar", "fr", "bn", "pt", "ru", "id", "ur", "de", "ja", "vi", "ko"} {
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
			if len(a) != len(b) {
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
