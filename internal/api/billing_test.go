package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/auth"
	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/pipeline"
	"github.com/githubflyideas/traffic66/internal/store"
	"github.com/githubflyideas/traffic66/internal/web"
)

// billingServer is a registered installation with a product line of two
// VIPs and an hour-old set of flows: 1000 in to A, 9000 out of A, 5000
// between A and B (inside the line), 2000 out of B — 17000 bytes once
// counted, 22000 if the members were simply added.
func billingServer(t *testing.T) (*Server, time.Time) {
	t.Helper()
	st, err := store.Open(store.Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ts := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Minute)
	a, b, user := "203.0.113.10", "203.0.113.11", "198.51.100.7"
	mk := func(src, dst string, wire uint64) store.Row {
		return store.Row{TS: ts, Src: src, Dst: dst, Client: src, Server: dst,
			Bytes: wire, Wire: wire, Pkts: 1, Flows: 1, Exporter: "10.0.0.1", Dir: store.DirTransit}
	}
	if err := st.AppendRows([]store.Row{mk(user, a, 1000), mk(a, user, 9000), mk(a, b, 5000), mk(b, user, 2000)}); err != nil {
		t.Fatal(err)
	}
	inv := enrich.NewInventory()
	if err := inv.Parse("product Shop vip=" + a + "," + b + "\n"); err != nil {
		t.Fatal(err)
	}
	s := registered(t)
	s.Store, s.Inv = st, inv
	s.DataDir = t.TempDir()
	s.Pipe = &pipeline.Pipeline{}
	s.Static = web.FS()
	s.Users = map[string]string{"admin": "pw"}
	s.sessions = map[string]session{}
	return s, ts
}

func billingWindow(ts time.Time) string {
	return fmt.Sprintf("from=%d&to=%d", ts.Add(-time.Minute).UnixMilli(), ts.Add(time.Hour).UnixMilli())
}

func TestBillingGateAndFigures(t *testing.T) {
	s, ts := billingServer(t)
	// an unregistered installation is refused, even one VIP group of two
	u := &Server{Store: s.Store, Inv: s.Inv}
	w := httptest.NewRecorder()
	u.billing(w, httptest.NewRequest("GET", "/api/billing?product=Shop&"+billingWindow(ts), nil))
	if w.Code != 403 || !strings.Contains(w.Body.String(), `"not_registered"`) {
		t.Fatalf("unregistered small group: %d %s", w.Code, w.Body)
	}
	// a subnet is a registered feature outright, refused with the
	// stable code
	w = httptest.NewRecorder()
	u.billing(w, httptest.NewRequest("GET", "/api/billing?subnet=203.0.113.0/24&"+billingWindow(ts), nil))
	if w.Code != 403 || !strings.Contains(w.Body.String(), `"not_registered"`) {
		t.Fatalf("unregistered subnet: %d %s", w.Code, w.Body)
	}
	// the product line's figures, counted once
	w = httptest.NewRecorder()
	s.billing(w, httptest.NewRequest("GET", "/api/billing?product=Shop&"+billingWindow(ts), nil))
	if w.Code != 200 {
		t.Fatalf("billing: %d %s", w.Code, w.Body)
	}
	var out billingOut
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Bytes != 17000 || out.InBytes != 6000 || out.OutBytes != 16000 {
		t.Errorf("bytes = %d (in %d, out %d), want 17000 (6000, 16000)", out.Bytes, out.InBytes, out.OutBytes)
	}
	if len(out.Members) != 2 {
		t.Errorf("members = %d, want 2", len(out.Members))
	}
	// one VIP alone, and the refusals
	w = httptest.NewRecorder()
	s.billing(w, httptest.NewRequest("GET", "/api/billing?vip=203.0.113.10&"+billingWindow(ts), nil))
	json.Unmarshal(w.Body.Bytes(), &out)
	if out.Bytes != 15000 {
		t.Errorf("single VIP bytes = %d, want 15000", out.Bytes)
	}
	for path, code := range map[string]string{
		"/api/billing?product=Nope&" + billingWindow(ts):      "unknown_product",
		"/api/billing?vip=203.0.113.999&" + billingWindow(ts): "bad_vip",
		"/api/billing?" + billingWindow(ts):                   "no_object",
	} {
		w = httptest.NewRecorder()
		s.billing(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 400 || !strings.Contains(w.Body.String(), `"`+code+`"`) {
			t.Errorf("%s: %d %s, want 400 %s", path, w.Code, w.Body, code)
		}
	}
	// the CSV carries the same figures
	w = httptest.NewRecorder()
	s.billingExport(w, httptest.NewRequest("GET", "/api/billing/export?product=Shop&"+billingWindow(ts), nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "total,,6000,16000,17000") {
		t.Errorf("export: %d %s", w.Code, w.Body)
	}
}

// readonlyServer is the billing fixture on a real password file: an
// administrator, one ordinary account, and room for read-only accounts
// the administrator makes through the API.
func readonlyServer(t *testing.T) (*Server, time.Time) {
	t.Helper()
	s, ts := billingServer(t)
	dir := t.TempDir()
	if err := auth.Set(dir, "admin", "admin-password-1"); err != nil {
		t.Fatal(err)
	}
	if err := auth.Set(dir, "alice", "alice-password-1"); err != nil {
		t.Fatal(err)
	}
	fc := auth.NewFileChecker(dir, nil)
	s.Accounts, s.Check, s.Exists, s.DataDir = fc, fc.Check, fc.Exists, dir
	s.Users = nil
	return s, ts
}

func TestReadonlyAccountScope(t *testing.T) {
	s, ts := readonlyServer(t)
	h := s.Handler()
	win := billingWindow(ts)
	call := func(method, path, body string, authz func(*http.Request)) *httptest.ResponseRecorder {
		var r *http.Request
		if body != "" {
			r = httptest.NewRequest(method, path, strings.NewReader(body))
		} else {
			r = httptest.NewRequest(method, path, nil)
		}
		if authz != nil {
			authz(r)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	basic := func(u, p string) func(*http.Request) {
		return func(r *http.Request) { r.SetBasicAuth(u, p) }
	}
	adminAuth := basic("admin", "admin-password-1")
	// no sign-in, no figures
	if w := call("GET", "/api/billing?product=Shop&"+win, "", nil); w.Code != 401 {
		t.Fatalf("signed out: %d, want 401", w.Code)
	}
	// an ordinary account cannot make read-only accounts; the admin can
	create := `{"user":"shop","password":"shop-password-1","kind":"product","object":"Shop"}`
	if w := call("POST", "/api/readonly", create, basic("alice", "alice-password-1")); w.Code != 403 {
		t.Fatalf("alice creating: %d %s, want 403", w.Code, w.Body)
	}
	if w := call("POST", "/api/readonly", create, adminAuth); w.Code != 200 {
		t.Fatalf("admin creating: %d %s", w.Code, w.Body)
	}
	if w := call("POST", "/api/readonly", create, adminAuth); w.Code != 409 {
		t.Fatalf("creating again: %d, want 409 exists", w.Code)
	}
	// sign in the ordinary way; the session knows the binding
	shopCookie := ""
	{
		w := call("POST", "/api/login", `{"user":"shop","password":"shop-password-1"}`, nil)
		if w.Code != 200 {
			t.Fatalf("shop login: %d %s", w.Code, w.Body)
		}
		for _, c := range w.Result().Cookies() {
			if c.Name == cookieName {
				shopCookie = c.Value
			}
		}
		if shopCookie == "" {
			t.Fatal("no session cookie")
		}
	}
	shop := func(r *http.Request) { r.AddCookie(&http.Cookie{Name: cookieName, Value: shopCookie}) }
	w := call("GET", "/api/me", "", shop)
	if !strings.Contains(w.Body.String(), `"readonly":true`) || !strings.Contains(w.Body.String(), `"object":"Shop"`) {
		t.Fatalf("me: %s", w.Body)
	}
	// the bound object, with the product line's figures, counted once —
	// asked for by name, and by asking for nothing in particular
	for _, path := range []string{"/api/billing?product=Shop&" + win, "/api/billing?" + win} {
		if w := call("GET", path, "", shop); w.Code != 200 || !strings.Contains(w.Body.String(), `"bytes":17000`) {
			t.Errorf("%s: %d %s, want 200 with 17000 bytes", path, w.Code, w.Body)
		}
	}
	// anything else is refused: other objects, other pages, every write
	for _, tc := range []struct{ method, path, body string }{
		{"GET", "/api/billing?vip=203.0.113.10&" + win, ""},
		{"GET", "/api/billing?product=Other&" + win, ""},
		{"GET", "/api/overview?range=24h", ""},
		{"GET", "/api/topn?range=24h", ""},
		{"GET", "/api/inventory", ""},
		{"GET", "/api/users", ""},
		{"POST", "/api/findings", `{}`},
		{"POST", "/api/password", `{"old":"shop-password-1","new":"another-password-1"}`},
		{"POST", "/api/readonly", create},
	} {
		if w := call(tc.method, tc.path, tc.body, shop); w.Code != 403 || !strings.Contains(w.Body.String(), `"readonly_scope"`) {
			t.Errorf("shop %s %s: %d %s, want 403 readonly_scope", tc.method, tc.path, w.Code, w.Body)
		}
	}
	// basic auth is held to the same scope
	if w := call("GET", "/api/overview?range=24h", "", basic("shop", "shop-password-1")); w.Code != 403 {
		t.Errorf("shop basic overview: %d, want 403", w.Code)
	}
	if w := call("GET", "/api/billing?product=Shop&"+win, "", basic("shop", "shop-password-1")); w.Code != 200 {
		t.Errorf("shop basic billing: %d %s, want 200", w.Code, w.Body)
	}
	// a window wider than fourteen days is refused, not cut; a whole
	// month is wider, so the month form is refused too
	wide := fmt.Sprintf("from=%d&to=%d", ts.Add(-20*24*time.Hour).UnixMilli(), ts.Add(time.Hour).UnixMilli())
	if w := call("GET", "/api/billing?product=Shop&"+wide, "", shop); w.Code != 400 || !strings.Contains(w.Body.String(), `"bad_range"`) {
		t.Errorf("wide window: %d %s, want 400 bad_range", w.Code, w.Body)
	}
	if w := call("GET", "/api/billing?product=Shop&month=2026-11", "", shop); w.Code != 400 {
		t.Errorf("month window: %d %s, want 400 bad_range", w.Code, w.Body)
	}
	// the ordinary account and the administrator are unchanged
	if w := call("GET", "/api/billing?product=Shop&"+win, "", basic("alice", "alice-password-1")); w.Code != 200 || !strings.Contains(w.Body.String(), `"bytes":17000`) {
		t.Errorf("alice billing: %d %s", w.Code, w.Body)
	}
	if w := call("GET", "/api/me", "", basic("alice", "alice-password-1")); w.Code != 200 || !strings.Contains(w.Body.String(), `"readonly":false`) {
		t.Errorf("alice me: %d %s", w.Code, w.Body)
	}
	// and no fourteen-day cap holds her: a whole month answers
	if w := call("GET", "/api/billing?product=Shop&month=2026-11", "", basic("alice", "alice-password-1")); w.Code != 200 {
		t.Errorf("alice month window: %d %s", w.Code, w.Body)
	}
	w = call("GET", "/api/readonly", "", adminAuth)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"user":"shop"`) || strings.Contains(w.Body.String(), "shop-password-1") {
		t.Errorf("readonly list: %d %s", w.Code, w.Body)
	}
	// rebinding moves the account to the VIP: the line is now refused
	upd := `{"user":"shop","kind":"vip","object":"203.0.113.10"}`
	if w := call("POST", "/api/readonly/update", upd, adminAuth); w.Code != 200 {
		t.Fatalf("rebind: %d %s", w.Code, w.Body)
	}
	if w := call("GET", "/api/billing?vip=203.0.113.10&"+win, "", shop); w.Code != 200 || !strings.Contains(w.Body.String(), `"bytes":15000`) {
		t.Errorf("rebound billing: %d %s, want 200 with 15000 bytes", w.Code, w.Body)
	}
	if w := call("GET", "/api/billing?product=Shop&"+win, "", shop); w.Code != 403 {
		t.Errorf("old binding after rebind: %d, want 403", w.Code)
	}
	// disabling ends sign-ins and the open session alike
	upd = `{"user":"shop","kind":"vip","object":"203.0.113.10","disabled":true}`
	if w := call("POST", "/api/readonly/update", upd, adminAuth); w.Code != 200 {
		t.Fatalf("disable: %d %s", w.Code, w.Body)
	}
	if w := call("POST", "/api/login", `{"user":"shop","password":"shop-password-1"}`, nil); w.Code != 403 || !strings.Contains(w.Body.String(), `"account_disabled"`) {
		t.Errorf("disabled login: %d %s, want 403 account_disabled", w.Code, w.Body)
	}
	// the open session was ended by the disabling itself, and basic
	// auth meets the disabled refusal head-on
	if w := call("GET", "/api/billing?"+win, "", shop); w.Code != 401 {
		t.Errorf("disabled session: %d, want 401 (the session ended)", w.Code)
	}
	if w := call("GET", "/api/billing?"+win, "", basic("shop", "shop-password-1")); w.Code != 403 || !strings.Contains(w.Body.String(), `"account_disabled"`) {
		t.Errorf("disabled basic auth: %d %s, want 403 account_disabled", w.Code, w.Body)
	}
	// a password reset by the administrator: the old password dies
	upd = `{"user":"shop","kind":"vip","object":"203.0.113.10","disabled":false}`
	call("POST", "/api/readonly/update", upd, adminAuth)
	if w := call("POST", "/api/readonly/password", `{"user":"shop","password":"shop-password-2"}`, adminAuth); w.Code != 200 {
		t.Fatalf("reset: %d %s", w.Code, w.Body)
	}
	if w := call("POST", "/api/login", `{"user":"shop","password":"shop-password-1"}`, nil); w.Code != 401 {
		t.Errorf("old password after reset: %d, want 401", w.Code)
	}
	if w := call("POST", "/api/login", `{"user":"shop","password":"shop-password-2"}`, nil); w.Code != 200 {
		t.Errorf("new password after reset: %d %s", w.Code, w.Body)
	}
	// deletion removes the account altogether
	if w := call("DELETE", "/api/readonly?user=shop", "", adminAuth); w.Code != 200 {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	if w := call("POST", "/api/login", `{"user":"shop","password":"shop-password-2"}`, nil); w.Code != 401 {
		t.Errorf("deleted account login: %d, want 401", w.Code)
	}
}

func TestReadonlyRateLimit(t *testing.T) {
	s, ts := readonlyServer(t)
	h := s.Handler()
	win := billingWindow(ts)
	mk := httptest.NewRequest("POST", "/api/readonly", strings.NewReader(
		`{"user":"metered","password":"metered-password-1","kind":"vip","object":"203.0.113.10"}`))
	mk.SetBasicAuth("admin", "admin-password-1")
	mw := httptest.NewRecorder()
	h.ServeHTTP(mw, mk)
	if mw.Code != 200 {
		t.Fatalf("create: %d %s", mw.Code, mw.Body)
	}
	for i := 0; i < roPerMinute; i++ {
		r := httptest.NewRequest("GET", "/api/billing?vip=203.0.113.10&"+win, nil)
		r.SetBasicAuth("metered", "metered-password-1")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("query %d: %d %s", i+1, w.Code, w.Body)
		}
	}
	r := httptest.NewRequest("GET", "/api/billing?vip=203.0.113.10&"+win, nil)
	r.SetBasicAuth("metered", "metered-password-1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 429 || !strings.Contains(w.Body.String(), `"rate_limited"`) {
		t.Fatalf("query 61: %d %s, want 429 rate_limited", w.Code, w.Body)
	}
	if ra, err := strconv.Atoi(w.Header().Get("Retry-After")); err != nil || ra < 1 || ra > 61 {
		t.Errorf("Retry-After = %q, want seconds until the window resets (1..61)", w.Header().Get("Retry-After"))
	}
}

func TestBillingAPIRules(t *testing.T) {
	s, ts := billingServer(t)
	h := s.Handler()
	call := func(method, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		r.SetBasicAuth("admin", "pw")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := call("POST", "/api/billing?product=Shop&"+billingWindow(ts)); w.Code != 405 {
		t.Errorf("POST /api/billing: %d, want 405", w.Code)
	}
	w := call("GET", "/api/v1/billing?product=Shop&"+billingWindow(ts))
	if w.Code != 200 || w.Header().Get("Api-Version") != "1" {
		t.Errorf("GET /api/v1/billing: %d, Api-Version %q", w.Code, w.Header().Get("Api-Version"))
	}
	// the no-sign-in share page of the first build is gone: nothing
	// answers there, and its data call neither
	for _, p := range []string{"/share", "/share/data?k=whatever"} {
		if w := call("GET", p); w.Code != 404 {
			t.Errorf("GET %s: %d, want 404", p, w.Code)
		}
	}
}
