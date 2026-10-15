package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/githubflyideas/traffic66/internal/auth"
)

// A new installation signs in with the default password, must change it
// before anything else, and its admin manages the other users.
func TestAccounts(t *testing.T) {
	dir := t.TempDir()
	if err := auth.Set(dir, "admin", auth.DefaultPassword); err != nil {
		t.Fatal(err)
	}
	fc := auth.NewFileChecker(dir, nil)
	s := &Server{Check: fc.Check, Exists: fc.Exists, Accounts: fc, DataDir: dir, sessions: map[string]session{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("GET /api/loginhint", s.loginHint)
	for p, h := range map[string]http.HandlerFunc{"GET /api/me": s.me, "POST /api/password": s.changePassword, "GET /api/users": s.listUsers,
		"POST /api/users": s.putUser, "DELETE /api/users": s.deleteUser, "GET /api/overview": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }} {
		mux.Handle(p, s.auth(h))
	}
	call := func(method, path, body, cookie string) (*httptest.ResponseRecorder, string) {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = "192.0.2.1:1"
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: cookieName, Value: cookie})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		for _, c := range w.Result().Cookies() {
			if c.Name == cookieName {
				cookie = c.Value
			}
		}
		return w, cookie
	}
	if w, _ := call("GET", "/api/loginhint", "", ""); !strings.Contains(w.Body.String(), `"default":true`) {
		t.Fatalf("hint %s", w.Body)
	}
	w, ck := call("POST", "/api/login", `{"user":"admin","password":"traffic66"}`, "")
	if !strings.Contains(w.Body.String(), `"must_change":true`) {
		t.Fatalf("login %s", w.Body)
	}
	if w, _ := call("GET", "/api/overview", "", ck); w.Code != 403 {
		t.Fatalf("overview before change: %d", w.Code)
	}
	if w, _ := call("POST", "/api/password", `{"old":"traffic66","new":"short"}`, ck); w.Code != 400 {
		t.Fatalf("short password: %d", w.Code)
	}
	if w, _ := call("POST", "/api/password", `{"old":"traffic66","new":"s3cret-pass"}`, ck); w.Code != 200 {
		t.Fatalf("change: %d %s", w.Code, w.Body)
	}
	if w, _ := call("GET", "/api/overview", "", ck); w.Code != 200 {
		t.Fatalf("overview after change: %d", w.Code)
	}
	if w, _ := call("GET", "/api/loginhint", "", ""); !strings.Contains(w.Body.String(), `"default":false`) {
		t.Fatalf("hint after change %s", w.Body)
	}
	// admin adds alice; alice cannot manage users
	if w, _ := call("POST", "/api/users", `{"user":"alice","password":"alice-pass"}`, ck); w.Code != 200 {
		t.Fatalf("add alice: %d %s", w.Code, w.Body)
	}
	_, ack := call("POST", "/api/login", `{"user":"alice","password":"alice-pass"}`, "")
	if w, _ := call("GET", "/api/users", "", ack); w.Code != 403 {
		t.Fatalf("alice lists users: %d", w.Code)
	}
	if w, _ := call("GET", "/api/me", "", ack); !strings.Contains(w.Body.String(), `"admin":false`) {
		t.Fatalf("alice me %s", w.Body)
	}
	if w, _ := call("DELETE", "/api/users?user=admin", "", ck); w.Code != 400 {
		t.Fatalf("admin deletes self: %d", w.Code)
	}
	if w, _ := call("DELETE", "/api/users?user=alice", "", ck); w.Code != 200 {
		t.Fatalf("delete alice: %d", w.Code)
	}
	if w, _ := call("GET", "/api/me", "", ack); w.Code != 401 {
		t.Fatalf("deleted alice still signed in: %d", w.Code)
	}
	// -password: nothing to change here
	fixed := &Server{Accounts: auth.NewFileChecker(dir, map[string]string{"admin": "x"}), DataDir: dir}
	rw := httptest.NewRecorder()
	fixed.changePassword(rw, httptest.NewRequest("POST", "/api/password", strings.NewReader(`{}`)))
	if rw.Code != 409 {
		t.Fatalf("fixed: %d", rw.Code)
	}
}

// An unregistered installation gets a few accounts, which is what makes a
// second and a third person possible without paying — and a fourth a
// reason to. The limit is on adding someone, never on the people already
// there.
func TestAccountLimit(t *testing.T) {
	dir := t.TempDir()
	const pw = "an-admin-password"
	if err := auth.Set(dir, "admin", pw); err != nil {
		t.Fatal(err)
	}
	fc := auth.NewFileChecker(dir, nil)
	s := &Server{Check: fc.Check, Exists: fc.Exists, Accounts: fc, DataDir: dir, sessions: map[string]session{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", s.login)
	for p, h := range map[string]http.HandlerFunc{"GET /api/users": s.listUsers, "POST /api/users": s.putUser} {
		mux.Handle(p, s.auth(h))
	}
	cookie := ""
	call := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = "192.0.2.1:1"
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: cookieName, Value: cookie})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		for _, c := range w.Result().Cookies() {
			if c.Name == cookieName {
				cookie = c.Value
			}
		}
		return w
	}
	call("POST", "/api/login", `{"user":"admin","password":"`+pw+`"}`)

	add := func(name string) *httptest.ResponseRecorder {
		return call("POST", "/api/users", `{"user":"`+name+`","password":"a-long-enough-password"}`)
	}
	// admin plus two more fills the free allowance of three.
	for _, name := range []string{"ops", "noc"} {
		if w := add(name); w.Code != 200 {
			t.Fatalf("adding %s: %d %s", name, w.Code, w.Body)
		}
	}
	w := add("fourth")
	if w.Code != http.StatusForbidden {
		t.Fatalf("the fourth account: %d %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), "user_limit") {
		t.Fatalf("the refusal should say why: %s", w.Body)
	}

	// Setting a password for somebody who is already here is not adding
	// anyone, so the limit must not touch it.
	if w := add("ops"); w.Code != 200 {
		t.Fatalf("changing an existing password: %d %s", w.Code, w.Body)
	}

	// The limit travels with the list, so the page can say so before
	// somebody types a name and a password for nothing.
	if w := call("GET", "/api/users", ""); !strings.Contains(w.Body.String(), `"limit":3`) {
		t.Fatalf("the list should carry the limit: %s", w.Body)
	}
}
