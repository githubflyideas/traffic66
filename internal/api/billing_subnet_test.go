package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/auth"
	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/store"
)

// capServer is an installation holding two product lines, of three
// and of four VIPs, with no traffic: the gate tests ask who may bill
// them, registered and not.
func capServer(t *testing.T) (reg, unreg *Server, ts time.Time) {
	t.Helper()
	st, err := store.Open(store.Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	inv := enrich.NewInventory()
	text := "product Trio vip=203.0.113.10,203.0.113.11,203.0.113.12\n" +
		"product Quad vip=203.0.113.20,203.0.113.21,203.0.113.22,203.0.113.23\n"
	if err := inv.Parse(text); err != nil {
		t.Fatal(err)
	}
	reg = registered(t)
	reg.Store, reg.Inv = st, inv
	unreg = &Server{Store: st, Inv: inv}
	return reg, unreg, time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Minute)
}

// Unregistered, the traffic statistics answer nothing: one VIP, a VIP
// group of any size and a subnet are all refused with not_registered,
// on the page's endpoint and the CSV alike, while a registered
// installation bills them.
func TestBillingUnregisteredRefused(t *testing.T) {
	reg, unreg, ts := capServer(t)
	win := billingWindow(ts)
	call := func(s *Server, path string, export bool) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		if export {
			s.billingExport(w, httptest.NewRequest("GET", path, nil))
		} else {
			s.billing(w, httptest.NewRequest("GET", path, nil))
		}
		return w
	}
	for _, q := range []string{"vip=203.0.113.10", "vip=203.0.113.10,203.0.113.11", "product=Trio", "product=Quad", "subnet=203.0.113.0/24"} {
		for _, export := range []bool{false, true} {
			w := call(unreg, "/api/billing?"+q+"&"+win, export)
			if w.Code != 403 || !strings.Contains(w.Body.String(), `"not_registered"`) {
				t.Errorf("unregistered %s (export=%v): %d %s, want 403 not_registered", q, export, w.Code, w.Body)
			}
		}
	}
	if w := call(reg, "/api/billing?product=Quad&"+win, false); w.Code != 200 {
		t.Errorf("registered Quad: %d %s, want 200", w.Code, w.Body)
	}
}

// subnetServer is a registered installation whose billing summaries
// hold one hour of flows: 1000 in to X, 9000 out of X, 5000 between
// X and Y inside 203.0.113.0/24, and 7000 in to Z outside it — the
// subnet reads 6000 in, 14000 out, 15000 counted once.
func subnetServer(t *testing.T) (*Server, time.Time) {
	t.Helper()
	st, err := store.Open(store.Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	hour := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Hour)
	const out, x, y, z = "198.51.100.7", "203.0.113.10", "203.0.113.11", "203.0.114.10"
	for _, f := range []struct {
		src, dst string
		wire     uint64
	}{{out, x, 1000}, {x, out, 9000}, {x, y, 5000}, {out, z, 7000}} {
		if _, err := st.DB.Exec(`INSERT INTO r_bill VALUES (?, false, ?, ?, ?)`, hour, f.src, f.dst, f.wire); err != nil {
			t.Fatal(err)
		}
	}
	s := registered(t)
	s.Store = st
	s.Inv = enrich.NewInventory()
	return s, hour
}

func TestBillingSubnet(t *testing.T) {
	s, hour := subnetServer(t)
	win := billingWindow(hour.Add(30 * time.Minute))
	// the figures, counted once, in the billing shape
	w := httptest.NewRecorder()
	s.billing(w, httptest.NewRequest("GET", "/api/billing?subnet=203.0.113.0/24&"+win, nil))
	if w.Code != 200 {
		t.Fatalf("subnet: %d %s", w.Code, w.Body)
	}
	var out billingOut
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Kind != "subnet" || out.Name != "203.0.113.0/24" {
		t.Errorf("object = %s %s, want subnet 203.0.113.0/24", out.Kind, out.Name)
	}
	if out.Bytes != 15000 || out.InBytes != 6000 || out.OutBytes != 14000 {
		t.Errorf("bytes = %d (in %d, out %d), want 15000 (6000, 14000)", out.Bytes, out.InBytes, out.OutBytes)
	}
	if len(out.VIPs) != 2 || out.VIPs[0] != "203.0.113.10" {
		t.Errorf("vips = %v, want the two member addresses, largest first", out.VIPs)
	}
	// an unmasked spelling asks for the range the address stands in
	w = httptest.NewRecorder()
	s.billing(w, httptest.NewRequest("GET", "/api/billing?subnet=203.0.113.10/24&"+win, nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"name":"203.0.113.0/24"`) ||
		!strings.Contains(w.Body.String(), `"bytes":15000`) {
		t.Errorf("unmasked: %d %s", w.Code, w.Body)
	}
	// the CSV carries the same figures
	w = httptest.NewRecorder()
	s.billingExport(w, httptest.NewRequest("GET", "/api/billing/export?subnet=203.0.113.0/24&"+win, nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "total,,6000,14000,15000") {
		t.Errorf("export: %d %s", w.Code, w.Body)
	}
	// a prefix that does not parse is a bad request with a stable code
	for _, bad := range []string{"203.0.113.0/33", "banana", "203.0.113.10"} {
		w = httptest.NewRecorder()
		s.billing(w, httptest.NewRequest("GET", "/api/billing?subnet="+bad+"&"+win, nil))
		if w.Code != 400 || !strings.Contains(w.Body.String(), `"bad_subnet"`) {
			t.Errorf("subnet=%s: %d %s, want 400 bad_subnet", bad, w.Code, w.Body)
		}
	}
}

// A read-only account is adjudicated by the installation's
// registration state like everyone else: bound to a line of four on
// an unregistered installation it meets the merged limit; bound to a
// line of two it is answered. And a subnet is outside its scope at
// any registration state.
func TestReadonlyBillingAdjudication(t *testing.T) {
	// the Shop fixture (a line of two) on an unregistered install
	s, ts := readonlyServer(t)
	s.License = nil
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
	adminAuth := func(r *http.Request) { r.SetBasicAuth("admin", "admin-password-1") }
	login := func(user, pw string) func(*http.Request) {
		w := call("POST", "/api/login", `{"user":"`+user+`","password":"`+pw+`"}`, nil)
		if w.Code != 200 {
			t.Fatalf("%s login: %d %s", user, w.Code, w.Body)
		}
		cookie := ""
		for _, c := range w.Result().Cookies() {
			if c.Name == cookieName {
				cookie = c.Value
			}
		}
		return func(r *http.Request) { r.AddCookie(&http.Cookie{Name: cookieName, Value: cookie}) }
	}
	create := `{"user":"shop","password":"shop-password-1","kind":"product","object":"Shop"}`
	if w := call("POST", "/api/readonly", create, adminAuth); w.Code != 200 {
		t.Fatalf("creating shop: %d %s", w.Code, w.Body)
	}
	shop := login("shop", "shop-password-1")
	// unregistered, a read-only account's own object is refused like
	// everything else on the page, and so is a subnet
	for _, q := range []string{"product=Shop", "subnet=203.0.113.0/24"} {
		if w := call("GET", "/api/billing?"+q+"&"+win, "", shop); w.Code != 403 ||
			!strings.Contains(w.Body.String(), `"not_registered"`) {
			t.Errorf("unregistered read-only %s: %d %s, want 403 not_registered", q, w.Code, w.Body)
		}
	}

	// a group of four on an unregistered install: the account is
	// created and signs in, and its queries are refused
	s2, ts2 := billingServer(t)
	s2.License = nil
	inv := enrich.NewInventory()
	if err := inv.Parse("product Quad vip=203.0.113.10,203.0.113.11,203.0.113.12,203.0.113.13\n"); err != nil {
		t.Fatal(err)
	}
	s2.Inv = inv
	dir := t.TempDir()
	if err := auth.Set(dir, "admin", "admin-password-1"); err != nil {
		t.Fatal(err)
	}
	fc := auth.NewFileChecker(dir, nil)
	s2.Accounts, s2.Check, s2.Exists, s2.DataDir = fc, fc.Check, fc.Exists, dir
	s2.Users = nil
	h2 := s2.Handler()
	call2 := func(method, path, body string, authz func(*http.Request)) *httptest.ResponseRecorder {
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
		h2.ServeHTTP(w, r)
		return w
	}
	create2 := `{"user":"quad","password":"quad-password-1","kind":"product","object":"Quad"}`
	if w := call2("POST", "/api/readonly", create2, adminAuth); w.Code != 200 {
		t.Fatalf("creating quad: %d %s", w.Code, w.Body)
	}
	w := call2("POST", "/api/login", `{"user":"quad","password":"quad-password-1"}`, nil)
	if w.Code != 200 {
		t.Fatalf("quad login: %d %s", w.Code, w.Body)
	}
	cookie := ""
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName {
			cookie = c.Value
		}
	}
	quad := func(r *http.Request) { r.AddCookie(&http.Cookie{Name: cookieName, Value: cookie}) }
	if w := call2("GET", "/api/billing?product=Quad&"+billingWindow(ts2), "", quad); w.Code != 403 ||
		!strings.Contains(w.Body.String(), `"not_registered"`) {
		t.Errorf("unregistered, bound group of four: %d %s, want 403 not_registered", w.Code, w.Body)
	}
}
