package api

import (
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubflyideas/traffic66/internal/enrich"
)

// A list imported from the page is saved where the startup loader reads
// it, and is in use at once.
func TestThreatListImport(t *testing.T) {
	dir := t.TempDir()
	s := &Server{Thr: enrich.NewThreats(), DataDir: dir}
	call := func(method, url, body string) int {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, url, strings.NewReader(body))
		if method == "POST" {
			s.putThreatList(w, r)
		} else {
			s.deleteThreatList(w, r)
		}
		return w.Code
	}
	if c := call("POST", "/api/threats/lists", `{"name":"drop","text":"# Spamhaus DROP\n198.51.100.0/24 ; SBL1\n203.0.113.66\n"}`); c != 200 {
		t.Fatal(c)
	}
	if got := s.Thr.Match(netip.MustParseAddr("198.51.100.7")); got != "drop" {
		t.Fatalf("not in use: %q", got)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "threats", "drop.txt")); err != nil || !strings.Contains(string(b), "203.0.113.66") {
		t.Fatalf("not saved: %v", err)
	}
	for _, bad := range []string{`{"name":"../etc","text":"1.2.3.4"}`, `{"name":"x","text":"no addresses here"}`, `{"name":"","text":"1.2.3.4"}`} {
		if c := call("POST", "/api/threats/lists", bad); c != 400 {
			t.Errorf("%s accepted: %d", bad, c)
		}
	}
	if c := call("DELETE", "/api/threats/lists?name=drop", ""); c != 200 {
		t.Fatal(c)
	}
	if s.Thr.Match(netip.MustParseAddr("198.51.100.7")) != "" || len(s.Thr.Lists()) != 0 {
		t.Fatal("the list is still in use")
	}
	if _, err := os.Stat(filepath.Join(dir, "threats", "drop.txt")); !os.IsNotExist(err) {
		t.Fatal("the file is still there")
	}
}
