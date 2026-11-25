package api

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// cookieFrom signs in over the given scheme and returns the session cookie.
func cookieFrom(t *testing.T, tlsOn bool, fwd string) *http.Cookie {
	t.Helper()
	s := &Server{Users: map[string]string{"admin": "right"}, sessions: map[string]session{}}
	r := httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"user":"admin","password":"right"}`))
	r.RemoteAddr = "192.0.2.5:40000"
	if tlsOn {
		r.TLS = &tls.ConnectionState{}
	}
	if fwd != "" {
		r.Header.Set("X-Forwarded-Proto", fwd)
	}
	w := httptest.NewRecorder()
	s.login(w, r)
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName {
			return c
		}
	}
	t.Fatalf("no session cookie; login returned %d", w.Code)
	return nil
}

// Over TLS the session cookie must be Secure, so it is never sent in clear
// if anything ever downgrades the connection.
func TestSessionCookieSecureOverTLS(t *testing.T) {
	if c := cookieFrom(t, true, ""); !c.Secure {
		t.Error("cookie is not Secure on a TLS connection")
	}
}

// Behind a proxy that terminates TLS the connection here is plain, and the
// header is the only way to know.
func TestSessionCookieSecureBehindProxy(t *testing.T) {
	for _, fwd := range []string{"https", "HTTPS", " https ", "https, http"} {
		if c := cookieFrom(t, false, fwd); !c.Secure {
			t.Errorf("X-Forwarded-Proto %q: cookie is not Secure", fwd)
		}
	}
}

// On a plain HTTP installation the cookie must NOT be Secure: the browser
// would drop it and nobody could sign in. traffic66 is run both ways.
func TestSessionCookieNotSecureOnPlainHTTP(t *testing.T) {
	for _, fwd := range []string{"", "http", "http, https"} {
		if c := cookieFrom(t, false, fwd); c.Secure {
			t.Errorf("X-Forwarded-Proto %q: cookie is Secure on a plain connection", fwd)
		}
	}
}

// The cookie keeps the attributes it already had.
func TestSessionCookieKeepsHttpOnlyAndSameSite(t *testing.T) {
	c := cookieFrom(t, true, "")
	if !c.HttpOnly {
		t.Error("cookie is not HttpOnly")
	}
	if c.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite is %v, want Strict", c.SameSite)
	}
}

// Signing out must actually clear the cookie it set: browsers compare
// Secure and Path when replacing one, so the clearing cookie has to match.
func TestLogoutClearsSecureCookie(t *testing.T) {
	s := &Server{Users: map[string]string{"admin": "right"}, sessions: map[string]session{}}
	r := httptest.NewRequest("POST", "/api/logout", nil)
	r.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()
	s.logout(w, r)
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName {
			if c.Value != "" || c.MaxAge >= 0 {
				t.Errorf("cookie not cleared: %+v", c)
			}
			if !c.Secure {
				t.Error("the clearing cookie is not Secure, so it does not replace the one that is")
			}
			return
		}
	}
	t.Fatal("logout set no cookie")
}
