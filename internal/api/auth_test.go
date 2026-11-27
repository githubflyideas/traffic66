package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/auth"
)

// TestLoginAnswersAlike: a login for a user that does not exist is
// answered exactly like a wrong password for one that does — same
// status, same body — and the checker does the same work for both
// (see auth.TestUnknownUserCostsTheSameWork), so neither the answer
// nor its timing names the accounts on an installation.
func TestLoginAnswersAlike(t *testing.T) {
	c := auth.NewChecker(nil, []auth.Entry{{User: "admin", Hash: auth.Hash("right")}})
	s := &Server{Check: c.Check}
	login := func(user, pw string) (int, string) {
		r := httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"user":"`+user+`","password":"`+pw+`"}`))
		r.RemoteAddr = "192.0.2.9:1"
		w := httptest.NewRecorder()
		s.login(w, r)
		return w.Code, w.Body.String()
	}
	codeKnown, bodyKnown := login("admin", "wrong")
	codeUnknown, bodyUnknown := login("nosuchuser", "wrong")
	if codeKnown != http.StatusUnauthorized || codeUnknown != codeKnown || bodyUnknown != bodyKnown {
		t.Errorf("known user, wrong password: %d %s; unknown user: %d %s", codeKnown, bodyKnown, codeUnknown, bodyUnknown)
	}
}

func TestFailedLoginsLockOut(t *testing.T) {
	now := time.Unix(1700000000, 0)
	s := &Server{Users: map[string]string{"admin": "right"}}
	s.fails.now = func() time.Time { return now }
	h := s.auth(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	try := func(ip, pw string) int {
		r := httptest.NewRequest("GET", "/api/status", nil)
		r.RemoteAddr = ip + ":40000"
		r.SetBasicAuth("admin", pw)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	for i := 0; i < maxFails; i++ {
		if c := try("192.0.2.9", "wrong"); c != 401 {
			t.Fatalf("attempt %d: %d", i, c)
		}
	}
	if c := try("192.0.2.9", "right"); c != 429 {
		t.Fatalf("locked client with the right password got %d, want 429", c)
	}
	if c := try("192.0.2.10", "right"); c != 200 {
		t.Fatalf("other client got %d", c)
	}
	now = now.Add(lockFor + time.Second)
	if c := try("192.0.2.9", "right"); c != 200 {
		t.Fatalf("after the lock expired: %d", c)
	}
	// The login endpoint shares the counter.
	for i := 0; i < maxFails; i++ {
		r := httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"user":"admin","password":"no"}`))
		r.RemoteAddr = "192.0.2.11:1"
		s.login(httptest.NewRecorder(), r)
	}
	r := httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"user":"admin","password":"right"}`))
	r.RemoteAddr = "192.0.2.11:1"
	w := httptest.NewRecorder()
	s.login(w, r)
	if w.Code != 429 {
		t.Fatalf("login after lockout: %d", w.Code)
	}
}
