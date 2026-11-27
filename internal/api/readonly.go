package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/githubflyideas/traffic66/internal/auth"
)

// Read-only accounts: a sign-in that exists to let a business check its
// own billing figures. The administrator makes one, bound to a single
// VIP or a single product line; the account signs in with a password
// like everyone else, and then the whole API except its bound object's
// traffic statistics is closed to it — every write included, even the
// ones ordinary accounts may make. There is no anonymous way in: a
// link with a key in it is the platform links' mechanism, for embedding
// a chart in another platform, and it is not reused for people.
//
// The password lives where every password lives, in the password file,
// hashed by the auth package and changed with the same machinery. Only
// the role lives here: readonly.json in the data directory, beside
// tokens.json, re-read when it changes, the way the API tokens are.
// That split is what lets the account sign in through the ordinary
// login with no second credential store to keep in step.

type roAccount struct {
	User     string    `json:"user"`
	Kind     string    `json:"kind"` // "vip" or "product"
	Object   string    `json:"object"`
	Disabled bool      `json:"disabled,omitempty"`
	Created  time.Time `json:"created"`
}

const readonlyFileName = "readonly.json"

type roStore struct {
	list   []roAccount
	stamp  string
	loaded bool
}

func (s *Server) roFile() string { return filepath.Join(s.DataDir, readonlyFileName) }

func (s *Server) loadReadonlyLocked(st *roStore) []roAccount {
	stamp := ""
	if fi, err := os.Stat(s.roFile()); err == nil {
		stamp = fi.ModTime().String() + strconv.FormatInt(fi.Size(), 10)
	}
	if !st.loaded || stamp != st.stamp {
		st.list = nil
		if b, err := os.ReadFile(s.roFile()); err == nil {
			if err := json.Unmarshal(b, &st.list); err != nil {
				s.audit("readonly file unreadable: %v", err)
				st.list = nil
			}
		}
		st.stamp, st.loaded = stamp, true
	}
	return st.list
}

func (s *Server) saveReadonlyLocked(st *roStore) error {
	if s.DataDir == "" {
		return nil // a server without a data directory keeps them in memory
	}
	b, err := json.MarshalIndent(st.list, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.roFile() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.roFile()); err != nil {
		return err
	}
	stamp := ""
	if fi, err := os.Stat(s.roFile()); err == nil {
		stamp = fi.ModTime().String() + strconv.FormatInt(fi.Size(), 10)
	}
	st.stamp, st.loaded = stamp, true
	return nil
}

// roRole returns the read-only role of a signed-in user, if they have
// one. It is looked up per request, so rebinding or disabling an
// account takes hold on the next call, not at the next sign-in.
func (s *Server) roRole(user string) (roAccount, bool) {
	if user == "" {
		return roAccount{}, false
	}
	s.roMu.Lock()
	defer s.roMu.Unlock()
	for _, ro := range s.loadReadonlyLocked(&s.roSt) {
		if ro.User == user {
			return ro, true
		}
	}
	return roAccount{}, false
}

// roAllowed is the gate every authenticated request passes when its
// user holds a read-only role: the statistics of the bound object, the
// caller's own description, and nothing else. It runs after the caller
// is known, on each of the session, basic-auth and token paths, so a
// read-only account is narrower than an ordinary one everywhere —
// including the writes ordinary accounts are allowed (marking a finding
// handled, changing their own password), which are refused here too.
func (s *Server) roAllowed(w http.ResponseWriter, r *http.Request, user string) bool {
	ro, ok := s.roRole(user)
	if !ok {
		return true
	}
	if ro.Disabled {
		writeErr(w, http.StatusForbidden, "account_disabled", nil)
		return false
	}
	switch r.Method + " " + r.URL.Path {
	case "GET /api/me", "GET /api/billing", "GET /api/billing/export":
		return true
	}
	writeErr(w, http.StatusForbidden, "readonly_scope", nil)
	return false
}

// roRate counts one query against an account, in a fixed minute
// window, on two dimensions: the account itself, and the account from
// this client address. Either one at the limit refuses the query.
type roRate struct {
	mu sync.Mutex
	m  map[string]*roWindow
}

type roWindow struct {
	start time.Time
	n     int
}

func (rl *roRate) allow(user, ip string, now time.Time) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.m == nil {
		rl.m = map[string]*roWindow{}
	}
	if len(rl.m) > 10000 {
		for k, w := range rl.m {
			if now.Sub(w.start) >= time.Minute {
				delete(rl.m, k)
			}
		}
	}
	ok, retry := true, 0
	for _, dim := range []string{user, user + "|" + ip} {
		w := rl.m[dim]
		if w == nil || now.Sub(w.start) >= time.Minute {
			w = &roWindow{start: now}
			rl.m[dim] = w
		}
		w.n++
		if w.n > roPerMinute {
			ok = false
			if r := int((time.Minute-now.Sub(w.start))/time.Second) + 1; r > retry {
				retry = r
			}
		}
	}
	return ok, retry
}

// endSessions drops every open session of one user: a reset password, a
// disabled account and a deleted one should not leave a signed-in tab
// behind them.
func (s *Server) endSessions(user string) {
	s.mu.Lock()
	for k, se := range s.sessions {
		if se.user == user {
			delete(s.sessions, k)
		}
	}
	s.mu.Unlock()
}

// listReadonly answers GET /api/readonly (administrators): the
// accounts and what each is bound to. No password material exists here
// to leak: the file holds roles, not credentials.
func (s *Server) listReadonly(w http.ResponseWriter, r *http.Request) {
	s.roMu.Lock()
	list := s.loadReadonlyLocked(&s.roSt)
	s.roMu.Unlock()
	out := make([]roAccount, 0, len(list))
	out = append(out, list...)
	writeJSON(w, http.StatusOK, map[string]any{"accounts": out,
		"registered": s.License != nil && s.License.State().Registered(), "per_minute": roPerMinute})
}

type roBody struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Kind     string `json:"kind"`
	Object   string `json:"object"`
	Disabled bool   `json:"disabled"`
}

func decodeRO(w http.ResponseWriter, r *http.Request) (roBody, bool) {
	var in roBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", nil)
		return in, false
	}
	in.User = strings.TrimSpace(in.User)
	return in, true
}

func validROUser(u string) bool {
	return u != "" && !strings.ContainsAny(u, ": \t\r\n") && !strings.HasPrefix(u, "#") && len(u) <= 64
}

// createReadonly answers POST /api/readonly: a new read-only account.
// The name must be free — converting an ordinary account by accident
// would take a person's console away, so an existing name is refused
// and the administrator deletes it first if that is really meant.
//
// The account counts against the installation's user limit like any
// other: it is an account in the password file, the limit is on how
// many of those an installation may have, and the ordinary creation
// path already counts it — so the check here is the same one, under
// the same lock, or the two paths together could pass the limit.
func (s *Server) createReadonly(w http.ResponseWriter, r *http.Request) {
	a := s.accounts(w)
	if a == nil {
		return
	}
	in, ok := decodeRO(w, r)
	if !ok {
		return
	}
	if !validROUser(in.User) {
		writeErr(w, http.StatusBadRequest, "bad_user", nil)
		return
	}
	if err := checkNew(in.Password); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	obj, err := s.resolveKindObject(in.Kind, in.Object)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	s.acctMu.Lock()
	if s.hasUser(in.User) {
		s.acctMu.Unlock()
		writeErr(w, http.StatusConflict, "exists", nil)
		return
	}
	if limit := s.userLimit(); limit > 0 && len(s.Accounts.Users()) >= limit {
		s.acctMu.Unlock()
		writeErr(w, http.StatusForbidden, "user_limit", map[string]any{"limit": limit})
		return
	}
	if err := auth.Set(s.DataDir, in.User, in.Password); err != nil {
		s.acctMu.Unlock()
		fail(w, err)
		return
	}
	s.acctMu.Unlock()
	ro := roAccount{User: in.User, Kind: obj.Kind, Object: obj.Name, Created: time.Now().UTC()}
	s.roMu.Lock()
	list := s.loadReadonlyLocked(&s.roSt)
	next := make([]roAccount, 0, len(list)+1)
	for _, cur := range list {
		if cur.User != in.User { // a stale role without an account is replaced
			next = append(next, cur)
		}
	}
	s.roSt.list = append(next, ro)
	err = s.saveReadonlyLocked(&s.roSt)
	s.roMu.Unlock()
	if err != nil {
		fail(w, err)
		return
	}
	_ = a
	s.audit("readonly account created user=%s bound=%s %s by=%s", ro.User, ro.Kind, ro.Object, userOf(r))
	writeJSON(w, http.StatusOK, map[string]any{"user": ro.User, "kind": ro.Kind, "object": ro.Object})
}

// updateReadonly answers POST /api/readonly/update: rebind an account
// to another object, or disable and re-enable it. A disabled account
// keeps its password and binding; it simply cannot sign in or query.
func (s *Server) updateReadonly(w http.ResponseWriter, r *http.Request) {
	if s.accounts(w) == nil {
		return
	}
	in, ok := decodeRO(w, r)
	if !ok {
		return
	}
	obj, err := s.resolveKindObject(in.Kind, in.Object)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	s.roMu.Lock()
	list := s.loadReadonlyLocked(&s.roSt)
	found := false
	for i, cur := range list {
		if cur.User == in.User {
			list[i].Kind, list[i].Object, list[i].Disabled = obj.Kind, obj.Name, in.Disabled
			found = true
		}
	}
	if found {
		err = s.saveReadonlyLocked(&s.roSt)
	}
	s.roMu.Unlock()
	if !found {
		writeErr(w, http.StatusNotFound, "no_user", nil)
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	if in.Disabled {
		s.endSessions(in.User)
	}
	s.audit("readonly account updated user=%s bound=%s %s disabled=%v by=%s", in.User, obj.Kind, obj.Name, in.Disabled, userOf(r))
	writeJSON(w, http.StatusOK, map[string]string{})
}

// resetReadonlyPassword answers POST /api/readonly/password: a new
// password for a read-only account, set by the administrator. The
// account's open sessions end, as with any reset.
func (s *Server) resetReadonlyPassword(w http.ResponseWriter, r *http.Request) {
	if s.accounts(w) == nil {
		return
	}
	in, ok := decodeRO(w, r)
	if !ok {
		return
	}
	if _, exists := s.roRole(in.User); !exists {
		writeErr(w, http.StatusNotFound, "no_user", nil)
		return
	}
	if err := checkNew(in.Password); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	s.acctMu.Lock()
	err := auth.Set(s.DataDir, in.User, in.Password)
	s.acctMu.Unlock()
	if err != nil {
		fail(w, err)
		return
	}
	s.endSessions(in.User)
	s.audit("readonly password reset user=%s by=%s", in.User, userOf(r))
	writeJSON(w, http.StatusOK, map[string]string{})
}

// deleteReadonly answers DELETE /api/readonly?user=…: the account and
// its role go together, and its sessions end.
func (s *Server) deleteReadonly(w http.ResponseWriter, r *http.Request) {
	if s.accounts(w) == nil {
		return
	}
	u := r.URL.Query().Get("user")
	if _, exists := s.roRole(u); !exists {
		writeErr(w, http.StatusNotFound, "no_user", nil)
		return
	}
	s.acctMu.Lock()
	err := auth.Delete(s.DataDir, u)
	s.acctMu.Unlock()
	if err != nil && !errors.Is(err, auth.ErrNoUser) {
		fail(w, err)
		return
	}
	s.roMu.Lock()
	list := s.loadReadonlyLocked(&s.roSt)
	next := make([]roAccount, 0, len(list))
	for _, cur := range list {
		if cur.User != u {
			next = append(next, cur)
		}
	}
	s.roSt.list = next
	err = s.saveReadonlyLocked(&s.roSt)
	s.roMu.Unlock()
	if err != nil {
		fail(w, err)
		return
	}
	s.dropTokensOf(u)
	s.endSessions(u)
	s.audit("readonly account deleted user=%s by=%s", u, userOf(r))
	writeJSON(w, http.StatusOK, map[string]string{})
}
