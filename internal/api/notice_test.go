package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/license"
	"github.com/githubflyideas/traffic66/internal/store"
)

// registered returns a server whose installation holds a valid licence.
func registered(t *testing.T) *Server {
	t.Helper()
	c, priv := signed(t, t.TempDir(), license.File{})
	s := &Server{License: c}
	text := sign(priv, license.File{Customer: "Example KK", Product: "traffic66",
		License: license.TypePerpetual, Tier: license.TierStandard, MaxUsers: 25})
	if w := install(t, s, text); w.Code != 200 {
		t.Fatalf("installing the licence: %d %s", w.Code, w.Body)
	}
	if !c.State().Registered() {
		t.Fatalf("not registered: %s", c.State().Reason)
	}
	return s
}

// Every API answer from an unregistered installation says so in a header,
// which is what a script that fetches the data sees.
func TestEvaluationHeaderOnAPIResponses(t *testing.T) {
	s := &Server{Users: map[string]string{"admin": "pw"}, sessions: map[string]session{}}
	h := s.auth(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	r := httptest.NewRequest("GET", "/api/status", nil)
	r.SetBasicAuth("admin", "pw")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if got := w.Header().Get(noticeHeader); got == "" {
		t.Error("no evaluation header on an unregistered installation")
	} else if !strings.Contains(got, "not registered") {
		t.Errorf("header says %q", got)
	}
}

// A registered installation's output is clean. This is the whole point of
// paying, so it is worth a test of its own.
func TestNoEvaluationHeaderWhenRegistered(t *testing.T) {
	s := registered(t)
	s.Users = map[string]string{"admin": "pw"}
	s.sessions = map[string]session{}
	h := s.auth(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	r := httptest.NewRequest("GET", "/api/status", nil)
	r.SetBasicAuth("admin", "pw")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if got := w.Header().Get(noticeHeader); got != "" {
		t.Errorf("a registered installation marked its output: %q", got)
	}
}

// An installation with no licence checker at all is the offline sandbox,
// and it is treated as unregistered: never more permissive than a missing
// licence.
func TestNoCheckerIsUnregistered(t *testing.T) {
	s := &Server{}
	if s.notice() == "" {
		t.Error("a server with no licence checker produced no notice")
	}
}

// The findings list is what gets pasted into a ticket, so the notice is in
// the body as well: copying the JSON keeps it, copying the headers does not.
func TestFindingsCarryNotice(t *testing.T) {
	st, err := store.Open(store.Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	now := time.Now().UTC().Truncate(time.Minute)
	st.SaveFinding(store.Finding{Kind: "scan", Sev: 3, Src: "10.1.1.1", Port: "445/tcp",
		First: now.Add(-20 * time.Minute), Last: now}, 30*time.Minute)

	body := func(s *Server) map[string]any {
		t.Helper()
		w := httptest.NewRecorder()
		s.findings(w, httptest.NewRequest("GET", "/api/findings?range=24h", nil))
		if w.Code != 200 {
			t.Fatalf("%d %s", w.Code, w.Body)
		}
		var out map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	out := body(&Server{Store: st})
	n, _ := out["notice"].(string)
	if !strings.Contains(n, "not registered") {
		t.Errorf("findings from an unregistered installation carry %q", n)
	}

	reg := registered(t)
	reg.Store = st
	if got := body(reg); got["notice"] != nil {
		t.Errorf("a registered installation marked its findings: %v", got["notice"])
	}
}

// The notice renders as comment lines for a text format.
func TestCommentLines(t *testing.T) {
	if got := commentLines("", "# "); got != "" {
		t.Errorf("a registered installation added %q", got)
	}
	got := commentLines("one\ntwo", "# ")
	if got != "# one\n# two\n" {
		t.Errorf("got %q", got)
	}
	// every line is commented, or a spreadsheet reads the rest as data
	for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if !strings.HasPrefix(line, "# ") {
			t.Errorf("line %q is not a comment", line)
		}
	}
}

// The notice itself must say what it is without needing the page's
// translations, because a file that has left has no language setting.
func TestNoticeWording(t *testing.T) {
	n := license.Unregistered("").Notice()
	for _, want := range []string{"EVALUATION", "traffic66", "not registered"} {
		if !strings.Contains(n, want) {
			t.Errorf("the notice does not mention %q: %q", want, n)
		}
	}
	if strings.Contains(n, "\n") {
		t.Error("the notice spans lines; it is used in single-line places too")
	}
}

// The asset CSV is the one file the API hands out, and it needs no notice
// because an unregistered installation cannot reach it at all: the ledger
// is a registered-only feature. This records that, so that whoever opens
// the gate knows the file leaves unmarked until they mark it.
func TestAssetCSVIsRegisteredOnly(t *testing.T) {
	s, _ := assetServer(t, nil, 0, 0, 0)
	r := httptest.NewRequest("GET", "/api/assets/csv?kind=devices", nil)
	r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, "admin"))
	w := httptest.NewRecorder()
	s.getAssetCSV(w, r)
	if w.Code != 403 {
		t.Fatalf("an unregistered installation got %d from the asset CSV, want 403; "+
			"if the gate has gone, the file now needs the notice on it", w.Code)
	}
}

// The asset CSV carries the notice as "# " comment lines when unregistered
// (defence in depth: today the route is registered-only) and none when
// registered.
func TestAssetCSVNotice(t *testing.T) {
	s, _ := assetServer(t, []string{license.FeatDCIM}, 0, 0, 0)
	r := httptest.NewRequest("GET", "/api/assets/csv?kind=site", nil)
	r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, "admin"))
	w := httptest.NewRecorder()
	s.getAssetCSV(w, r)
	if w.Code != 200 {
		t.Fatalf("registered export: %d %s", w.Code, w.Body)
	}
	if strings.HasPrefix(w.Body.String(), "#") {
		t.Errorf("a registered installation marked its CSV: %q", w.Body.String())
	}
	if got := commentLines((&Server{}).notice(), "# "); !strings.HasPrefix(got, "# ") || !strings.Contains(got, "not registered") {
		t.Errorf("unregistered CSV prefix = %q", got)
	}
}
