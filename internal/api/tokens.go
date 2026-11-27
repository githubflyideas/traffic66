package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/githubflyideas/traffic66/internal/auth"
)

// The API tokens live in tokens.json in the data directory, next to the
// password file. The server keeps them in memory and re-reads the file
// when it changes, so a token made with `traffic66 gen-token` on the
// command line works without a restart — and one revoked there stops.

func (s *Server) tokenFile() string { return filepath.Join(s.DataDir, auth.TokensFileName) }

// loadTokensLocked refreshes the in-memory list when the file changed.
// The caller holds tokMu.
func (s *Server) loadTokensLocked() []auth.Token {
	stamp := ""
	if fi, err := os.Stat(s.tokenFile()); err == nil {
		stamp = fmt.Sprint(fi.ModTime().UnixNano(), fi.Size())
	}
	if !s.tokLoaded || stamp != s.tokStamp {
		toks, err := auth.LoadTokens(s.DataDir)
		if err != nil {
			s.audit("token file unreadable: %v", err)
			toks = nil
		}
		s.toks, s.tokStamp, s.tokLoaded = toks, stamp, true
	}
	return s.toks
}

func (s *Server) tokens() []auth.Token {
	s.tokMu.Lock()
	defer s.tokMu.Unlock()
	return s.loadTokensLocked()
}

// tokenAuth finds the token a secret belongs to and notes the use. A
// token whose user no longer exists is worth nothing.
func (s *Server) tokenAuth(secret string) (auth.Token, bool) {
	for _, t := range s.tokens() {
		if !t.Matches(secret) {
			continue
		}
		if s.Exists != nil && !s.Exists(t.User) {
			return auth.Token{}, false
		}
		if time.Since(t.LastUsed) > time.Minute {
			s.tokMu.Lock()
			for i, cur := range s.loadTokensLocked() {
				if cur.ID == t.ID {
					s.toks[i].LastUsed = time.Now().UTC()
				}
			}
			auth.SaveTokens(s.DataDir, s.toks)
			s.tokMu.Unlock()
		}
		return t, true
	}
	return auth.Token{}, false
}

// tokenView is a token as the account page shows it: everything but the
// hash.
type tokenView struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	User     string    `json:"user"`
	Scope    string    `json:"scope"`
	Created  time.Time `json:"created"`
	LastUsed time.Time `json:"last_used,omitempty"`
}

func viewOf(t auth.Token) tokenView {
	return tokenView{ID: t.ID, Name: t.Name, User: t.User, Scope: t.Scope, Created: t.Created, LastUsed: t.LastUsed}
}

func (s *Server) listTokens(w http.ResponseWriter, r *http.Request) {
	var out []tokenView
	for _, t := range s.tokens() {
		out = append(out, viewOf(t))
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": out})
}

// createToken makes a token for a user (the caller, unless an
// administrator names another) and shows its secret — this once.
func (s *Server) createToken(w http.ResponseWriter, r *http.Request) {
	var in struct{ Name, Scope, User string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", nil)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 64 {
		writeErr(w, http.StatusBadRequest, "bad_name", nil)
		return
	}
	if in.Scope == "" {
		in.Scope = "read"
	}
	user := in.User
	if user == "" {
		user = userOf(r)
	}
	if s.Accounts != nil && !s.hasUser(user) {
		writeErr(w, http.StatusNotFound, "no_user", nil)
		return
	}
	tok, secret, err := auth.NewToken(user, in.Name, in.Scope)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_scope", nil)
		return
	}
	s.tokMu.Lock()
	s.toks = append(s.loadTokensLocked(), tok)
	err = auth.SaveTokens(s.DataDir, s.toks)
	s.tokMu.Unlock()
	if err != nil {
		fail(w, err)
		return
	}
	s.audit("token created name=%q user=%s scope=%s by=%s", tok.Name, tok.User, tok.Scope, userOf(r))
	writeJSON(w, http.StatusOK, map[string]any{"token": secret, "entry": viewOf(tok)})
}

// deleteToken revokes a token by id.
func (s *Server) deleteToken(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	s.tokMu.Lock()
	var kept []auth.Token
	var gone *auth.Token
	for i, t := range s.loadTokensLocked() {
		if t.ID == id {
			gone = &s.toks[i]
			continue
		}
		kept = append(kept, t)
	}
	if gone == nil {
		s.tokMu.Unlock()
		writeErr(w, http.StatusNotFound, "no_token", nil)
		return
	}
	s.toks = kept
	err := auth.SaveTokens(s.DataDir, s.toks)
	s.tokMu.Unlock()
	if err != nil {
		fail(w, err)
		return
	}
	s.audit("token revoked name=%q user=%s by=%s", gone.Name, gone.User, userOf(r))
	writeJSON(w, http.StatusOK, map[string]string{})
}

// dropTokensOf deletes every API token of a user. A token belongs to an
// account, not to a name: without this, the next account made under the
// same name would be what the old token opens.
func (s *Server) dropTokensOf(user string) {
	s.tokMu.Lock()
	defer s.tokMu.Unlock()
	var keep []auth.Token
	n := 0
	for _, t := range s.loadTokensLocked() {
		if t.User == user {
			n++
			continue
		}
		keep = append(keep, t)
	}
	if n == 0 {
		return
	}
	if err := auth.SaveTokens(s.DataDir, keep); err != nil {
		// the account is gone either way, and a token whose user does not
		// exist opens nothing until a new account takes the name; say so
		s.audit("tokens of deleted user %s could not be removed: %v", user, err)
		return
	}
	s.toks, s.tokLoaded = keep, true
	s.audit("tokens of deleted user %s removed (%d)", user, n)
}
