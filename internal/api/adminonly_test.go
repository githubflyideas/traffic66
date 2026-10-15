package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/githubflyideas/traffic66/internal/auth"
)

// openWrites are the calls that change something and are deliberately
// open to everyone who can sign in, with the reason. Everything else
// that writes has to go through adminAPI, and TestEveryWriteIsDecided
// fails when a new route is added without that decision being made.
var openWrites = map[string]string{
	"POST /api/login":           "signing in is how you stop being anonymous",
	"POST /api/logout":          "ending one's own session",
	"POST /api/password":        "one's own password; the old one is asked for again",
	"POST /api/users":           "admin()'s own gate, which also refuses a fixed password file",
	"DELETE /api/users":         "same",
	"POST /api/resolve":         "reverse DNS for the names on screen; stores nothing",
	"POST /api/findings":        "marking an alert handled is the everyday work, not administration",
	"POST /api/sandbox/files":   "offline analysis: uploading a capture to look at",
	"POST /api/sandbox/active":  "choosing which uploaded capture to look at",
	"POST /api/assets/csv":      "ledgerAdmin: the licence gate and the admin check together",
	"POST /api/assets/paste":    "same",
	"POST /api/assets/{kind}":   "same",
	"DELETE /api/assets/{kind}": "same",
}

// TestEveryWriteIsDecided reads the route table and fails on a writing
// route that is neither behind adminAPI nor listed in openWrites.
//
// It parses the source because that is the only place the whole table is
// visible: http.ServeMux will not list what was registered with it. The
// bug this guards against is not a wrong gate but a missing one — a
// route added next to the others, inheriting a wrapper that only asks
// whether the caller is signed in — so the test has to see the table as
// it is written rather than the handlers one by one.
func TestEveryWriteIsDecided(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "server.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	gated, open := map[string]bool{}, map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok || len(c.Args) == 0 {
			return true
		}
		fn, ok := c.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		lit, ok := c.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		pat, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		switch fn.Name {
		case "adminAPI":
			gated[pat] = true
		case "api":
			open[pat] = true
		}
		return true
	})
	if len(gated) == 0 || len(open) == 0 {
		t.Fatalf("read no route table: %d gated, %d open", len(gated), len(open))
	}
	// mux.HandleFunc routes are registered by name, not through either
	// wrapper; they are in openWrites and checked by the same rule.
	ast.Inspect(f, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok || len(c.Args) == 0 {
			return true
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "HandleFunc" && sel.Sel.Name != "Handle" {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "mux" {
			return true
		}
		if lit, ok := c.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if pat, err := strconv.Unquote(lit.Value); err == nil {
				open[pat] = true
			}
		}
		return true
	})
	var missing []string
	for pat := range open {
		method, _, ok := strings.Cut(pat, " ")
		if !ok || method == "GET" || method == "HEAD" {
			continue
		}
		if openWrites[pat] == "" {
			missing = append(missing, pat)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("these routes change something but ask only whether the caller is signed in:\n\t%s\n"+
			"Register each with adminAPI, or add it to openWrites with the reason it is open to everyone.",
			strings.Join(missing, "\n\t"))
	}
	// A stale entry is worth hearing about too: it reads as a decision
	// that has been made when the route is gone.
	for pat := range openWrites {
		if !open[pat] && !gated[pat] {
			t.Errorf("openWrites lists %q, which the route table no longer has", pat)
		}
		if gated[pat] {
			t.Errorf("openWrites lists %q, which is behind adminAPI; drop the entry", pat)
		}
	}
}

// TestAdminOnly: a signed-in user who is not the administrator is
// refused, the administrator is let through, and so is this machine's
// own token, which the TUI carries.
func TestAdminOnly(t *testing.T) {
	dir := t.TempDir()
	for _, u := range [][2]string{{"admin", "admin-pass"}, {"alice", "alice-pass"}} {
		if err := auth.Set(dir, u[0], u[1]); err != nil {
			t.Fatal(err)
		}
	}
	fc := auth.NewFileChecker(dir, nil)
	s := &Server{Check: fc.Check, Exists: fc.Exists, Accounts: fc, DataDir: dir,
		LocalTok: "local-token", sessions: map[string]session{}}
	reached := false
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", s.login)
	mux.Handle("POST /gated", s.auth(s.adminOnly(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})))
	login := func(user, pass string) string {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"user":"`+user+`","password":"`+pass+`"}`))
		r.RemoteAddr = "192.0.2.1:1"
		mux.ServeHTTP(w, r)
		for _, c := range w.Result().Cookies() {
			if c.Name == cookieName {
				return c.Value
			}
		}
		t.Fatalf("no session for %s: %d %s", user, w.Code, w.Body)
		return ""
	}
	post := func(cookie, bearer string) int {
		reached = false
		r := httptest.NewRequest("POST", "/gated", nil)
		r.RemoteAddr = "192.0.2.1:1"
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: cookieName, Value: cookie})
		}
		if bearer != "" {
			r.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w.Code
	}
	if code := post(login("alice", "alice-pass"), ""); code != http.StatusForbidden {
		t.Errorf("alice: got %d, want 403", code)
	}
	if reached {
		t.Error("alice reached the handler")
	}
	if code := post(login("admin", "admin-pass"), ""); code != http.StatusOK {
		t.Errorf("admin: got %d, want 200", code)
	}
	if !reached {
		t.Error("admin did not reach the handler")
	}
	if code := post("", "local-token"); code != http.StatusOK {
		t.Errorf("local token: got %d, want 200", code)
	}
	if !reached {
		t.Error("the local token did not reach the handler")
	}
	if code := post("", "wrong-token"); code != http.StatusUnauthorized {
		t.Errorf("wrong token: got %d, want 401", code)
	}
}

// TestAdminOnlyWhenPasswordsAreFixed is the regression this gate could
// easily have been: -password keeps the accounts in memory and leaves no
// password file to edit, so admin() answers 409 "fixed". Gating on that
// would lock such an installation out of its own settings.
func TestAdminOnlyWhenPasswordsAreFixed(t *testing.T) {
	fc := auth.NewFileChecker("", map[string]string{"admin": "admin-pass"})
	if !fc.Fixed() {
		t.Fatal("expected a fixed checker")
	}
	s := &Server{Check: fc.Check, Exists: fc.Exists, Accounts: fc, sessions: map[string]session{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", s.login)
	mux.Handle("POST /gated", s.auth(s.adminOnly(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"user":"admin","password":"admin-pass"}`))
	r.RemoteAddr = "192.0.2.1:1"
	mux.ServeHTTP(w, r)
	var ck string
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName {
			ck = c.Value
		}
	}
	if ck == "" {
		t.Fatalf("no session: %d %s", w.Code, w.Body)
	}
	r = httptest.NewRequest("POST", "/gated", nil)
	r.RemoteAddr = "192.0.2.1:1"
	r.AddCookie(&http.Cookie{Name: cookieName, Value: ck})
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("admin with a fixed password file: got %d %s, want 200", w.Code, w.Body)
	}
}
