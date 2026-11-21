package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/githubflyideas/traffic66/internal/auth"
	"github.com/githubflyideas/traffic66/internal/pipeline"
)

// The family's API rules, end to end through the real handler: a version
// header and a /v1 spelling, JSON errors with stable codes, 405 with an
// Allow header on a wrong method, scoped bearer tokens, an error for an
// unknown range, and a user limit that parallel requests cannot slip past.
func apiRulesServer(t *testing.T) (*Server, http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	const pw = "admin-pass-123"
	if err := auth.Set(dir, "admin", pw); err != nil {
		t.Fatal(err)
	}
	if err := auth.RecordAdmin(dir, "admin"); err != nil {
		t.Fatal(err)
	}
	fc := auth.NewFileChecker(dir, nil)
	s := &Server{Check: fc.Check, Exists: fc.Exists, Accounts: fc, DataDir: dir, Pipe: &pipeline.Pipeline{}}
	return s, s.Handler(), pw
}

func apiCall(h http.Handler, method, path, body, cookie, bearer string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr = "192.0.2.1:1"
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: cookieName, Value: cookie})
	}
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func apiLogin(t *testing.T, h http.Handler, pw string) string {
	t.Helper()
	w := apiCall(h, "POST", "/api/login", `{"user":"admin","password":"`+pw+`"}`, "", "")
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName {
			return c.Value
		}
	}
	t.Fatal("login set no session cookie")
	return ""
}

func TestAPIVersionAndPrefix(t *testing.T) {
	_, h, pw := apiRulesServer(t)
	ck := apiLogin(t, h, pw)
	w := apiCall(h, "GET", "/api/me", "", ck, "")
	if w.Code != 200 {
		t.Fatalf("me: %d %s", w.Code, w.Body)
	}
	if got := w.Header().Get("Api-Version"); got != apiVersion {
		t.Fatalf("Api-Version = %q, want %q", got, apiVersion)
	}
	w = apiCall(h, "GET", "/api/v1/me", "", ck, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"user":"admin"`) {
		t.Fatalf("v1 me: %d %s", w.Code, w.Body)
	}
}

func TestAPIMethodNotAllowedAndJSON404(t *testing.T) {
	_, h, pw := apiRulesServer(t)
	ck := apiLogin(t, h, pw)
	w := apiCall(h, "POST", "/api/me", `{}`, ck, "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /api/me: %d %s", w.Code, w.Body)
	}
	if allow := w.Header().Get("Allow"); !strings.Contains(allow, "GET") {
		t.Fatalf("Allow = %q", allow)
	}
	if !strings.Contains(w.Body.String(), `"error":"method_not_allowed"`) {
		t.Fatalf("405 body: %s", w.Body)
	}
	w = apiCall(h, "PUT", "/api/users", `{}`, ck, "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT /api/users: %d %s", w.Code, w.Body)
	}
	for _, m := range []string{"GET", "POST", "DELETE"} {
		if !strings.Contains(w.Header().Get("Allow"), m) {
			t.Fatalf("Allow = %q, missing %s", w.Header().Get("Allow"), m)
		}
	}
	w = apiCall(h, "GET", "/api/no-such-thing", "", ck, "")
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), `"error":"not_found"`) {
		t.Fatalf("unknown path: %d %s", w.Code, w.Body)
	}
}

func TestAPITokens(t *testing.T) {
	s, h, pw := apiRulesServer(t)
	mk := func(user, scope string) string {
		tok, secret, err := auth.NewToken(user, "test-"+scope, scope)
		if err != nil {
			t.Fatal(err)
		}
		toks, _ := auth.LoadTokens(s.DataDir)
		if err := auth.SaveTokens(s.DataDir, append(toks, tok)); err != nil {
			t.Fatal(err)
		}
		return secret
	}
	read, write := mk("admin", "read"), mk("admin", "write")
	ghost := mk("ghost", "read")

	if w := apiCall(h, "GET", "/api/me", "", "", read); w.Code != 200 || !strings.Contains(w.Body.String(), `"user":"admin"`) {
		t.Fatalf("read token me: %d %s", w.Code, w.Body)
	}
	if w := apiCall(h, "POST", "/api/password", `{"old":"`+pw+`","new":"another-pass-9"}`, "", read); w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "token_scope") {
		t.Fatalf("read token write: %d %s", w.Code, w.Body)
	}
	if w := apiCall(h, "GET", "/api/me", "", "", "t66_bogus"); w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "token_invalid") {
		t.Fatalf("bogus token: %d %s", w.Code, w.Body)
	}
	if w := apiCall(h, "GET", "/api/me", "", "", ghost); w.Code != http.StatusUnauthorized {
		t.Fatalf("token of a deleted user: %d %s", w.Code, w.Body)
	}
	// last: the write token may do what its user may, here change the password
	if w := apiCall(h, "POST", "/api/password", `{"old":"`+pw+`","new":"another-pass-9"}`, "", write); w.Code != 200 {
		t.Fatalf("write token change password: %d %s", w.Code, w.Body)
	}
}

func TestBadRangeIsAnError(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/overview?range=bogus", nil)
	if _, err := parseQuery(r); err != errBadRange {
		t.Fatalf("range=bogus: err = %v", err)
	}
	r = httptest.NewRequest("GET", "/api/overview?range=30m", nil)
	if _, err := parseQuery(r); err != nil {
		t.Fatalf("range=30m: err = %v", err)
	}
	w := httptest.NewRecorder()
	fail(w, errBadRange)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"error":"bad_range"`) {
		t.Fatalf("fail(bad_range): %d %s", w.Code, w.Body)
	}
}

// The v2.6.6 finding: the free limit of three accounts could be passed by
// creating users in parallel, because the check and the write were not
// under one lock. Twenty parallel creates must land exactly two.
func TestUserLimitRace(t *testing.T) {
	_, h, pw := apiRulesServer(t)
	ck := apiLogin(t, h, pw)
	var wg sync.WaitGroup
	codes := make([]int, 20)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body := `{"user":"racer` + strings.Repeat("x", i) + `","password":"a-long-enough-password"}`
			codes[i] = apiCall(h, "POST", "/api/users", body, ck, "").Code
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, c := range codes {
		if c == 200 {
			ok++
		}
	}
	if ok != 2 {
		t.Fatalf("parallel creates landed %d, want 2 (limit 3 incl. admin); codes %v", ok, codes)
	}
	w := apiCall(h, "GET", "/api/users", "", ck, "")
	if n := strings.Count(w.Body.String(), `"racer`); n != 2 {
		t.Fatalf("user list after race: %s", w.Body)
	}
}
