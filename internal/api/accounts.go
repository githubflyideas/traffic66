package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/githubflyideas/traffic66/internal/auth"
)

// loginHint tells the sign-in page whether to show the default password:
// always in the demo, and while some user of a new installation still has it.
func (s *Server) loginHint(w http.ResponseWriter, r *http.Request) {
	def := s.Demo || (s.Accounts != nil && s.Accounts.HasDefault())
	out := map[string]any{"demo": s.Demo, "default": def}
	if def {
		out["user"], out["password"] = "admin", auth.DefaultPassword
	}
	writeJSON(w, http.StatusOK, out)
}

// me describes the signed-in user: whether they manage the others, whether
// they must change the default password, and whether passwords can be
// changed here at all (not when given with -password).
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u := userOf(r)
	out := map[string]any{"user": u, "admin": false, "must_change": false, "fixed": true, "ldap": false}
	if s.Accounts != nil {
		out["fixed"] = s.Accounts.Fixed()
		out["admin"] = u != "" && u == s.Accounts.Admin()
	}
	if c, err := r.Cookie(cookieName); err == nil {
		s.mu.Lock()
		out["must_change"] = s.sessions[c.Value].mustChange
		s.mu.Unlock()
	}
	writeJSON(w, http.StatusOK, out)
}

// accounts returns the password file when it can be changed from here.
func (s *Server) accounts(w http.ResponseWriter) *auth.FileChecker {
	if s.Accounts == nil || s.Accounts.Fixed() || s.DataDir == "" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "fixed"})
		return nil
	}
	return s.Accounts
}

func checkNew(pw string) error {
	if len([]rune(pw)) < auth.MinLength {
		return errors.New("too_short")
	}
	if pw == auth.DefaultPassword {
		return errors.New("default")
	}
	return nil
}

// changePassword sets the signed-in user's own password; the old one is
// asked for again.
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	a := s.accounts(w)
	if a == nil {
		return
	}
	var in struct{ Old, New string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	u := userOf(r)
	if u == "" || !s.checkPassword(u, in.Old) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "wrong_old"})
		return
	}
	if err := checkNew(in.New); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := auth.Set(s.DataDir, u, in.New); err != nil {
		fail(w, err)
		return
	}
	s.mu.Lock()
	for k, se := range s.sessions {
		if se.user == u {
			se.mustChange = false
			s.sessions[k] = se
		}
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{})
}

// admin answers 403 unless the signed-in user manages the others.
func (s *Server) admin(w http.ResponseWriter, r *http.Request) *auth.FileChecker {
	a := s.accounts(w)
	if a == nil {
		return nil
	}
	if u := userOf(r); u == "" || u != a.Admin() {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not_admin"})
		return nil
	}
	return a
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	a := s.admin(w, r)
	if a == nil {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": a.Users(), "admin": a.Admin()})
}

// putUser adds a user, or sets a new password for one.
func (s *Server) putUser(w http.ResponseWriter, r *http.Request) {
	if s.admin(w, r) == nil {
		return
	}
	var in struct{ User, Password string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	in.User = strings.TrimSpace(in.User)
	if in.User == "" || strings.ContainsAny(in.User, ": \t\r\n") || len(in.User) > 64 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_user"})
		return
	}
	if err := checkNew(in.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := auth.Set(s.DataDir, in.User, in.Password); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{})
}

// deleteUser removes a user; their open sessions end. The administrator
// cannot remove themselves.
func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	a := s.admin(w, r)
	if a == nil {
		return
	}
	u := r.URL.Query().Get("user")
	if u == userOf(r) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "self"})
		return
	}
	if err := auth.Delete(s.DataDir, u); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{})
}
