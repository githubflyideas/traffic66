// Package api serves the JSON API and the embedded web UI.
package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"github.com/githubflyideas/traffic66/internal/snmp"
	"io/fs"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/githubflyideas/traffic66/internal/auth"
	"github.com/githubflyideas/traffic66/internal/collector"
	"github.com/githubflyideas/traffic66/internal/detect"
	"github.com/githubflyideas/traffic66/internal/dnsres"
	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/license"
	"github.com/githubflyideas/traffic66/internal/pipeline"
	"github.com/githubflyideas/traffic66/internal/sandbox"
	"github.com/githubflyideas/traffic66/internal/store"
)

// Server wires the API together.
type Server struct {
	Store   *store.Store
	Pipe    *pipeline.Pipeline
	Col     *collector.Collector
	Inv     *enrich.Inventory
	ASN     *enrich.ASNDB
	Thr     *enrich.Threats
	DNS     *dnsres.Resolver
	Det     *detect.Detector
	SB      *sandbox.Sandbox // offline analysis of capture files
	Offline bool             // started on capture files (traffic66 file.pcap): no live data
	License *license.Checker // trial or licence state for the page footer
	// AutoLogin is a one-time token: /auto?t=<it> signs the browser in once.
	AutoLogin string
	Static    fs.FS
	Version   string
	Demo      bool
	Users     map[string]string          // fixed passwords (tests)
	Check     func(user, pw string) bool // login check; replaces Users when set
	Exists    func(user string) bool     // whether a user still exists; signed-in sessions of deleted users end
	LocalTok  string                     // token for the TUI on this machine
	// Accounts is the password file: changing passwords and managing users
	// from the web UI. Nil in tests and when it does not apply.
	Accounts *auth.FileChecker
	Capture   func() []CaptureInfo
	SNMP      func() []snmp.Status
	Started   time.Time
	DataDir   string

	mu       sync.Mutex
	sessions map[string]session
	fails    failLimiter
	rates    rates
}

type session struct {
	user string
	exp  time.Time
	// mustChange: signed in with the default password; only changing it
	// is allowed until then
	mustChange bool
}

type ctxKey struct{}

// userOf returns the signed-in user of a request ("" for the TUI token).
func userOf(r *http.Request) string {
	u, _ := r.Context().Value(ctxKey{}).(string)
	return u
}

// allowedBeforeChange are the calls a session that must change its
// password may make.
var allowedBeforeChange = map[string]bool{"/api/me": true, "/api/password": true, "/api/status": true}

// CaptureInfo describes a local capture interface.
type CaptureInfo struct {
	Iface   string `json:"iface"`
	Method  string `json:"method"`
	Packets uint64 `json:"packets"`
	Dropped uint64 `json:"dropped"`
	Flows   int    `json:"active_flows"`
	Err     string `json:"error,omitempty"`
}

const cookieName = "t66s"

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	s.sessions = map[string]session{}
	if s.Started.IsZero() {
		s.Started = time.Now()
	}
	go s.sampleRates()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("POST /api/logout", s.logout)
	mux.HandleFunc("GET /auto", s.autoLogin)
	mux.HandleFunc("GET /logo", s.logo)
	mux.HandleFunc("GET /api/loginhint", s.loginHint)
	api := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, s.auth(h)) }
	api("GET /api/status", s.status)
	api("GET /api/me", s.me)
	api("POST /api/password", s.changePassword)
	api("GET /api/users", s.listUsers)
	api("POST /api/users", s.putUser)
	api("DELETE /api/users", s.deleteUser)
	api("GET /api/overview", s.data((*Server).overview))
	api("GET /api/topn", s.data((*Server).topn))
	api("GET /api/sankey", s.data((*Server).sankey))
	api("GET /api/series", s.data((*Server).series))
	api("GET /api/records", s.data((*Server).records))
	api("GET /api/threats", s.data((*Server).threats))
	api("GET /api/rings", s.data((*Server).rings))
	api("GET /api/geolines", s.data((*Server).geoLines))
	api("GET /api/ifaces", s.ifaces)
	api("GET /api/recon", s.recon)
	api("GET /api/sources", s.sources)
	api("POST /api/resolve", s.resolve)
	api("GET /api/inventory", s.getInventory)
	api("POST /api/inventory", s.putInventory)
	api("POST /api/iface", s.putIface)
	api("GET /api/findings", s.data((*Server).findings))
	api("POST /api/findings", s.data((*Server).setFindings))
	api("GET /api/cleanup", s.getCleanup)
	api("POST /api/cleanup", s.postCleanup)
	api("GET /api/sandbox", s.getSandbox)
	api("POST /api/sandbox/files", s.putSandboxFile)
	api("DELETE /api/sandbox/files", s.deleteSandbox)
	api("POST /api/sandbox/active", s.putSandboxActive)
	api("POST /api/logo", s.putLogo)
	api("DELETE /api/logo", s.deleteLogo)
	api("GET /api/geo", s.getGeo)
	api("POST /api/geo", s.putGeo)
	api("DELETE /api/geo", s.deleteGeo)
	api("POST /api/geo/dbip", s.updateDBIP)
	static := http.FileServer(http.FS(s.Static))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		// no-cache on every file: after an upgrade the browser must not keep
		// the old app.js or app.css (the embedded files carry no date to
		// revalidate against, so a cached copy could outlive the program)
		w.Header().Set("Cache-Control", "no-cache")
		static.ServeHTTP(w, r)
	})
	return mux
}

// failLimiter locks out a client address after repeated failed logins.
type failLimiter struct {
	mu  sync.Mutex
	m   map[string]*failState
	now func() time.Time
}

type failState struct {
	n           int
	first, lock time.Time
}

const (
	maxFails   = 5
	failWindow = time.Minute
	lockFor    = time.Minute
)

func clientIP(r *http.Request) string {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return h
}

func (f *failLimiter) clock() time.Time {
	if f.now != nil {
		return f.now()
	}
	return time.Now()
}

// blocked reports whether ip is locked out.
func (f *failLimiter) blocked(ip string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	st := f.m[ip]
	return st != nil && f.clock().Before(st.lock)
}

// fail records a failed attempt.
func (f *failLimiter) fail(ip string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.clock()
	if f.m == nil {
		f.m = map[string]*failState{}
	}
	if len(f.m) > 10000 { // forget stale entries
		for k, v := range f.m {
			if now.Sub(v.first) > failWindow && now.After(v.lock) {
				delete(f.m, k)
			}
		}
	}
	st := f.m[ip]
	if st == nil || now.Sub(st.first) > failWindow {
		st = &failState{first: now}
		f.m[ip] = st
	}
	st.n++
	if st.n >= maxFails {
		st.lock = now.Add(lockFor)
		st.n, st.first = 0, now
	}
}

func (f *failLimiter) ok(ip string) {
	f.mu.Lock()
	delete(f.m, ip)
	f.mu.Unlock()
}

func tooMany(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many failed logins; try again in a minute"})
}

func (s *Server) auth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); tok != "" && s.LocalTok != "" &&
			subtle.ConstantTimeCompare([]byte(tok), []byte(s.LocalTok)) == 1 {
			next(w, r)
			return
		}
		if u, p, ok := r.BasicAuth(); ok {
			ip := clientIP(r)
			if s.fails.blocked(ip) {
				tooMany(w)
				return
			}
			if !s.checkPassword(u, p) {
				s.fails.fail(ip)
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login"})
				return
			}
			s.fails.ok(ip)
			next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
			return
		}
		if c, err := r.Cookie(cookieName); err == nil {
			s.mu.Lock()
			se, ok := s.sessions[c.Value]
			if ok && s.Exists != nil && !s.Exists(se.user) {
				delete(s.sessions, c.Value)
				ok = false
			}
			if ok && time.Now().Before(se.exp) {
				se.exp = time.Now().Add(12 * time.Hour)
				s.sessions[c.Value] = se
				s.mu.Unlock()
				if se.mustChange && !allowedBeforeChange[r.URL.Path] {
					writeJSON(w, http.StatusForbidden, map[string]string{"error": "change_password"})
					return
				}
				next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, se.user)))
				return
			}
			s.mu.Unlock()
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login"})
	})
}

func (s *Server) checkPassword(u, p string) bool {
	if s.Check != nil {
		return s.Check(u, p)
	}
	want, ok := s.Users[u]
	return ok && subtle.ConstantTimeCompare([]byte(want), []byte(p)) == 1
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ User, Password string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	ip := clientIP(r)
	if s.fails.blocked(ip) {
		tooMany(w)
		return
	}
	if !s.checkPassword(in.User, in.Password) {
		s.fails.fail(ip)
		time.Sleep(500 * time.Millisecond)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "wrong user or password"})
		return
	}
	s.fails.ok(ip)
	b := make([]byte, 24)
	rand.Read(b)
	tok := hex.EncodeToString(b)
	s.mu.Lock()
	now := time.Now()
	for k, v := range s.sessions {
		if now.After(v.exp) {
			delete(s.sessions, k)
		}
	}
	// the default password of a new installation is changed before
	// anything else (not in the demo, nor with -password)
	must := !s.Demo && s.Accounts != nil && !s.Accounts.Fixed() && in.Password == auth.DefaultPassword
	s.sessions[tok] = session{user: in.User, exp: now.Add(12 * time.Hour), mustChange: must}
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: tok, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
	writeJSON(w, http.StatusOK, map[string]any{"user": in.User, "must_change": must})
}

// autoLogin signs in with the one-time token printed for traffic66 file.pcap
// and opens the offline analysis.
func (s *Server) autoLogin(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("t")
	s.mu.Lock()
	ok := s.AutoLogin != "" && subtle.ConstantTimeCompare([]byte(t), []byte(s.AutoLogin)) == 1
	if ok {
		s.AutoLogin = "" // once
		b := make([]byte, 24)
		rand.Read(b)
		tok := hex.EncodeToString(b)
		s.sessions[tok] = session{user: "admin", exp: time.Now().Add(12 * time.Hour)}
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: tok, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}
	s.mu.Unlock()
	http.Redirect(w, r, "/#v=sandbox", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		s.mu.Lock()
		delete(s.sessions, c.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]string{})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("api: encode: %v", err)
	}
}

// duckErr matches DuckDB's error format ("Out of Memory Error: …").
var duckErr = regexp.MustCompile(`\b[A-Z][A-Za-z]* Error: `)

// fail answers a failed request. Mistakes in the request (a bad filter)
// are returned as they are. Database failures are logged in full and the
// browser gets only their kind, so pages can say "the database is at its
// memory limit" instead of showing SQL.
func fail(w http.ResponseWriter, err error) {
	msg := err.Error()
	kind := ""
	switch {
	case strings.Contains(msg, "Out of Memory") || strings.Contains(msg, "memory_limit") || strings.Contains(msg, "failed to allocate"):
		kind = "memory"
	case duckErr.MatchString(msg) || strings.Contains(msg, "database") || strings.Contains(msg, "driver:"):
		kind = "storage"
	}
	if kind == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	log.Printf("api: database (%s): %v", kind, err)
	w.Header().Set("Retry-After", "30")
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database " + kind, "kind": kind})
}

// rates samples pipeline counters once per second.
type rates struct {
	mu                 sync.Mutex
	recPerSec, rowRate float64
	lastRec, lastRow   uint64
}

func (s *Server) sampleRates() {
	t := time.NewTicker(time.Second)
	n := 0
	for now := range t.C {
		if n++; n%5 == 0 {
			s.sampleSources(now)
		}
		rec, row := s.Pipe.Records.Load(), s.Pipe.Rows.Load()
		s.rates.mu.Lock()
		s.rates.recPerSec = s.rates.recPerSec*0.8 + float64(rec-s.rates.lastRec)*0.2
		s.rates.rowRate = s.rates.rowRate*0.8 + float64(row-s.rates.lastRow)*0.2
		s.rates.lastRec, s.rates.lastRow = rec, row
		s.rates.mu.Unlock()
	}
}
