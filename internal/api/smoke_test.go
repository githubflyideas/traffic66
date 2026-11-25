package api

import (
	"encoding/json"

	"github.com/githubflyideas/traffic66/internal/auth"
	"github.com/githubflyideas/traffic66/internal/collector"
	"github.com/githubflyideas/traffic66/internal/detect"
	"github.com/githubflyideas/traffic66/internal/dnsres"
	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/pipeline"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every page's data call, through the whole handler as a browser or a
// program reaches it: signed in, each answers 200 with JSON (or CSV for
// the exports), and each still answers when there is little or no data.
// The handlers are tested one by one elsewhere; this is the sweep that
// catches a route that panics, answers HTML, or lost its gate.
func TestEveryDataRouteAnswers(t *testing.T) {
	s, ts := billingServer(t)
	wire(s)
	h := s.Handler()
	win := billingWindow(ts)
	call := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.SetBasicAuth("admin", "pw")
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, p := range []string{
		"/api/status", "/api/me", "/api/openapi.json", "/api/v1/status",
		"/api/overview?range=24h", "/api/overview?" + win,
		"/api/topn?range=24h&dim=client&by=wire", "/api/topn?range=7d&dim=server&by=bytes",
		"/api/sankey?range=24h", "/api/series?range=24h&by=client",
		"/api/records?range=1h", "/api/threats?range=24h",
		"/api/rings?range=24h&inner=server&outer=client", "/api/geolines?range=24h",
		"/api/ifaces?range=24h", "/api/recon?range=24h&exporter=10.0.0.1&ifindex=1", "/api/sources",
		"/api/inventory", "/api/racks", "/api/findings?range=24h",
		"/api/cleanup", "/api/geo", "/api/tokens", "/api/users", "/api/readonly",
		"/api/billing?product=Shop&" + win,
	} {
		w := call("GET", p, "")
		if w.Code != 200 {
			t.Errorf("GET %s: %d %.200s", p, w.Code, w.Body)
			continue
		}
		if !json.Valid(w.Body.Bytes()) {
			t.Errorf("GET %s: not JSON: %.120s", p, w.Body)
		}
		if w.Header().Get("Api-Version") == "" {
			t.Errorf("GET %s: no Api-Version header", p)
		}
	}
	for _, p := range []string{"/api/export/rollups?range=24h", "/api/billing/export?product=Shop&" + win} {
		w := call("GET", p, "")
		if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/csv") ||
			!strings.Contains(w.Header().Get("Content-Disposition"), ".csv") {
			t.Errorf("GET %s: %d %s %s", p, w.Code, w.Header().Get("Content-Type"), w.Body)
		}
	}
	if w := call("GET", "/healthz", ""); w.Code != 200 {
		t.Errorf("healthz: %d", w.Code)
	}
	// the same routes refuse a caller who has not signed in
	for _, p := range []string{"/api/overview?range=24h", "/api/tokens", "/api/export/rollups?range=24h"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("GET %s unsigned: %d, want 401", p, w.Code)
		}
	}
}

// An API token is made, used within its scope, and removed.
func TestTokenLifecycle(t *testing.T) {
	s, _ := billingServer(t)
	wire(s)
	h := s.Handler()
	do := func(method, path, body, bearer string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if bearer != "" {
			r.Header.Set("Authorization", "Bearer "+bearer)
		} else {
			r.SetBasicAuth("admin", "pw")
		}
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	w := do("POST", "/api/tokens", `{"name":"grafana","scope":"read"}`, "")
	if w.Code != 200 {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	var made map[string]any
	json.Unmarshal(w.Body.Bytes(), &made)
	tok, _ := made["token"].(string)
	if !strings.HasPrefix(tok, "t66_") {
		t.Fatalf("token %q", tok)
	}
	if w := do("GET", "/api/overview?range=1h", "", tok); w.Code != 200 {
		t.Errorf("read token reading: %d", w.Code)
	}
	if w := do("POST", "/api/inventory", `{"text":""}`, tok); w.Code != http.StatusForbidden {
		t.Errorf("read token writing: %d, want 403", w.Code)
	}
	if w := do("GET", "/api/tokens", "", ""); !strings.Contains(w.Body.String(), "grafana") || strings.Contains(w.Body.String(), tok) {
		t.Errorf("list shows the name, never the secret: %s", w.Body)
	}
	entry, _ := made["entry"].(map[string]any)
	id, _ := entry["id"].(string)
	if w := do("DELETE", "/api/tokens?id="+id, "", ""); w.Code != 200 {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	if w := do("GET", "/api/overview?range=1h", "", tok); w.Code != http.StatusUnauthorized {
		t.Errorf("deleted token: %d, want 401", w.Code)
	}
}

// wire gives a test server the parts main always gives it.
func wire(s *Server) {
	s.ASN = enrich.NewASNDB()
	s.Thr = enrich.NewThreats()
	s.Pipe = pipeline.New(pipeline.Config{}, s.Store, s.Inv, s.ASN, s.Thr)
	s.Col = collector.New(s.Pipe)
	s.DNS = dnsres.New(dnsres.Options{PerSecond: 1e-9, Upstream: "127.0.0.1:1"})
	s.Det = detect.New(s.Store, s.Inv, detect.Config{})
	// an administrator "admin" with the password "pw", from the account
	// files as a real installation has them
	if err := auth.Set(s.DataDir, "admin", "pw"); err != nil {
		panic(err)
	}
	if err := auth.RecordAdmin(s.DataDir, "admin"); err != nil {
		panic(err)
	}
	fc := auth.NewFileChecker(s.DataDir, nil)
	s.Check, s.Exists, s.Accounts, s.Users = fc.Check, fc.Exists, fc, nil
}

// The two writes an administrator makes most: the inventory, which is
// saved whole or not at all, and purging one device's records, which
// asks for an explicit confirmation.
func TestInventoryAndPurge(t *testing.T) {
	s, ts := billingServer(t)
	wire(s)
	inv, err := enrich.LoadInventory(filepath.Join(s.DataDir, "inventory.txt"))
	if err != nil {
		t.Fatal(err)
	}
	inv.Parse(s.Inv.Text())
	s.Inv = inv
	h := s.Handler()
	call := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.SetBasicAuth("admin", "pw")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	good := `{"text":"device 10.0.0.1 Core\r\nproduct Shop vip=203.0.113.10,203.0.113.11\r\n"}`
	if w := call("POST", "/api/inventory", good); w.Code != 200 {
		t.Fatalf("save: %d %s", w.Code, w.Body)
	}
	if s.Inv.Device(netip.MustParseAddr("10.0.0.1")) != "Core" {
		t.Error("saved inventory not in effect")
	}
	if b, _ := os.ReadFile(filepath.Join(s.DataDir, "inventory.txt")); strings.Contains(string(b), "\r") || !strings.Contains(string(b), "device 10.0.0.1 Core") {
		t.Errorf("file on disk: %q", b)
	}
	if w := call("POST", "/api/inventory", `{"text":"device nope X\n"}`); w.Code < 400 {
		t.Errorf("bad inventory saved: %d", w.Code)
	}
	if s.Inv.Device(netip.MustParseAddr("10.0.0.1")) != "Core" {
		t.Error("a refused save replaced the inventory")
	}

	total := func() float64 {
		w := call("GET", "/api/billing?product=Shop&"+billingWindow(ts), "")
		var out billingOut
		json.Unmarshal(w.Body.Bytes(), &out)
		return float64(out.Bytes)
	}
	if total() == 0 {
		t.Fatal("no traffic before the purge")
	}
	if w := call("POST", "/api/cleanup/exporter", `{"exporter":"10.0.0.1"}`); w.Code != 400 {
		t.Errorf("purge without confirm: %d", w.Code)
	}
	if w := call("POST", "/api/cleanup/exporter", `{"exporter":"not-an-address","confirm":true}`); w.Code != 400 {
		t.Errorf("purge of a bad address: %d", w.Code)
	}
	if total() == 0 {
		t.Fatal("a refused purge deleted records")
	}
	if w := call("POST", "/api/cleanup/exporter", `{"exporter":"10.0.0.1","confirm":true}`); w.Code != 200 {
		t.Fatalf("purge: %d %s", w.Code, w.Body)
	}
	if got := total(); got != 0 {
		t.Errorf("after purging the only device: %v bytes, want 0", got)
	}
	if w := call("POST", "/api/cleanup", `{"days":-1}`); w.Code != 400 {
		t.Errorf("cleanup with negative days: %d", w.Code)
	}
	if w := call("POST", "/api/cleanup", `{}`); w.Code != 400 {
		t.Errorf("cleanup without days: %d", w.Code)
	}
}
