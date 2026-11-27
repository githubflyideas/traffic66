package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseTrustedProxies(t *testing.T) {
	tp, err := ParseTrustedProxies("10.0.0.1, 192.168.0.0/16 ,2001:db8::1")
	if err != nil {
		t.Fatal(err)
	}
	for _, ip := range []string{"10.0.0.1", "192.168.7.9", "2001:db8::1"} {
		if !tp.trusted(ip) {
			t.Errorf("%s should be trusted", ip)
		}
	}
	for _, ip := range []string{"10.0.0.2", "192.169.0.1", "1.1.1.1", "", "junk"} {
		if tp.trusted(ip) {
			t.Errorf("%s should not be trusted", ip)
		}
	}
	if tp, err := ParseTrustedProxies(""); err != nil || tp.trusted("10.0.0.1") {
		t.Errorf("empty list: err=%v, trusts 10.0.0.1=%v", err, tp.trusted("10.0.0.1"))
	}
	for _, bad := range []string{"example.org", "10.0.0.0/33", "10.0.0.1, nope"} {
		if _, err := ParseTrustedProxies(bad); err == nil {
			t.Errorf("%q: want an error", bad)
		}
	}
}

// TestClientIPForwardedFor: the client a request is accounted to is
// the first X-Forwarded-For address only when the direct peer is a
// trusted proxy; from anyone else the header is ignored.
func TestClientIPForwardedFor(t *testing.T) {
	tp, err := ParseTrustedProxies("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{TrustedProxies: tp}
	for _, tc := range []struct {
		remote, xff, want string
	}{
		{"10.1.2.3:9000", "1.1.1.1, 10.9.9.9", "1.1.1.1"}, // trusted proxy: first in the chain
		{"10.1.2.3:9000", "", "10.1.2.3"},                 // trusted, nothing forwarded
		{"10.1.2.3:9000", "not-an-ip", "10.1.2.3"},        // trusted, unusable header
		{"192.0.2.5:9000", "1.1.1.1", "192.0.2.5"},        // untrusted peer: header ignored
		{"192.0.2.5:9000", "", "192.0.2.5"},
	} {
		r := httptest.NewRequest("GET", "/api/status", nil)
		r.RemoteAddr = tc.remote
		if tc.xff != "" {
			r.Header.Set("X-Forwarded-For", tc.xff)
		}
		if got := s.clientIP(r); got != tc.want {
			t.Errorf("remote %s xff %q: clientIP = %s, want %s", tc.remote, tc.xff, got, tc.want)
		}
	}
	// Nobody trusted (the default): the header never counts.
	s = &Server{}
	r := httptest.NewRequest("GET", "/api/status", nil)
	r.RemoteAddr = "10.1.2.3:9000"
	r.Header.Set("X-Forwarded-For", "1.1.1.1")
	if got := s.clientIP(r); got != "10.1.2.3" {
		t.Errorf("no trusted proxies: clientIP = %s, want the peer 10.1.2.3", got)
	}
}

// lockoutServer is two users behind the clock the test controls.
func lockoutServer(t *testing.T, proxies string) (*Server, http.Handler) {
	t.Helper()
	tp, err := ParseTrustedProxies(proxies)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1700000000, 0)
	s := &Server{Users: map[string]string{"alice": "alicepw", "bob": "bobpw"}, TrustedProxies: tp}
	s.fails.now = func() time.Time { return now }
	return s, s.auth(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
}

func basicFrom(h http.Handler, remote, xff, user, pw string) int {
	r := httptest.NewRequest("GET", "/api/status", nil)
	r.RemoteAddr = remote + ":40000"
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	r.SetBasicAuth(user, pw)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code
}

// TestLockoutIsPerAccount: failures against one account never lock
// another account out, not even from the same address.
func TestLockoutIsPerAccount(t *testing.T) {
	_, h := lockoutServer(t, "")
	for i := 0; i < maxFails; i++ {
		if c := basicFrom(h, "192.0.2.9", "", "alice", "wrong"); c != 401 {
			t.Fatalf("alice attempt %d: %d", i, c)
		}
	}
	if c := basicFrom(h, "192.0.2.9", "", "alice", "alicepw"); c != 429 {
		t.Errorf("alice, locked: got %d, want 429", c)
	}
	if c := basicFrom(h, "192.0.2.9", "", "bob", "bobpw"); c != 200 {
		t.Errorf("bob from the same address: got %d, want 200", c)
	}
	if c := basicFrom(h, "192.0.2.10", "", "alice", "alicepw"); c != 200 {
		t.Errorf("alice from another address: got %d, want 200", c)
	}
}

// TestLockoutBehindTrustedProxy: behind a trusted proxy the clients
// are the forwarded addresses, so one user's failures lock out only
// that user at that address — not everybody behind the proxy.
func TestLockoutBehindTrustedProxy(t *testing.T) {
	_, h := lockoutServer(t, "10.0.0.1")
	for i := 0; i < maxFails; i++ {
		if c := basicFrom(h, "10.0.0.1", "1.1.1.1", "alice", "wrong"); c != 401 {
			t.Fatalf("alice attempt %d: %d", i, c)
		}
	}
	if c := basicFrom(h, "10.0.0.1", "1.1.1.1", "alice", "alicepw"); c != 429 {
		t.Errorf("alice at her own address: got %d, want 429", c)
	}
	if c := basicFrom(h, "10.0.0.1", "2.2.2.2", "bob", "bobpw"); c != 200 {
		t.Errorf("bob behind the same proxy: got %d, want 200", c)
	}
	if c := basicFrom(h, "10.0.0.1", "2.2.2.2", "alice", "alicepw"); c != 200 {
		t.Errorf("alice from another address: got %d, want 200", c)
	}
	// The same requests through a peer that is not trusted are all
	// one client — the peer — whatever the header claims.
	_, h2 := lockoutServer(t, "")
	for i := 0; i < maxFails; i++ {
		basicFrom(h2, "10.0.0.1", "1.1.1.1", "alice", "wrong")
	}
	if c := basicFrom(h2, "10.0.0.1", "2.2.2.2", "alice", "alicepw"); c != 429 {
		t.Errorf("untrusted peer, forged header: got %d, want 429 (the peer is the client)", c)
	}
}
