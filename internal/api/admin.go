package api

import "net/http"

// isAdmin reports whether the signed-in user manages the others. With no
// password file there is no administrator, so nothing may be written by
// anyone.
func (s *Server) isAdmin(r *http.Request) bool {
	// This machine's own token counts: the TUI and the local subcommands
	// carry it, and whoever can read it from the data directory could
	// have changed the files there directly.
	if local(r) {
		return true
	}
	if s.Accounts == nil {
		return false
	}
	u := userOf(r)
	return u != "" && u == s.Accounts.Admin()
}

// adminOnly guards h with requireAdmin.
func (s *Server) adminOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.requireAdmin(w, r) {
			h(w, r)
		}
	}
}

// requireAdmin answers 403 unless the caller may change the installation
// itself, and is the gate behind every call that deletes stored data or
// reconfigures the installation for everybody.
//
// It is deliberately not admin(): that one asks for a password file it
// can write and answers 409 "fixed" without one, which is right when the
// call is about the account list and wrong here. An installation started
// with -password, or without a data directory, still has an
// administrator — it simply cannot add users — so gating these calls on
// admin() would lock such an installation out of its own settings.
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if s.isAdmin(r) {
		return true
	}
	writeJSON(w, http.StatusForbidden, map[string]string{"error": "not_admin"})
	return false
}
