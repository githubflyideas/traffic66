package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"

	"github.com/githubflyideas/traffic66/internal/license"
	"github.com/githubflyideas/traffic66/internal/syslogout"
)

// The syslog feed, set from the findings page: where findings are sent,
// in which format, and how it is going. Sending is for registered
// installations; reading the settings is not, so the page can show what
// would be sent and why it is not.

// syslogFile is where the settings are kept.
func (s *Server) syslogFile() string { return filepath.Join(s.DataDir, "syslog.json") }

func (s *Server) syslogAllowed() bool {
	return s.License != nil && s.License.State().Allowed(license.FeatSyslog)
}

func (s *Server) getSyslog(w http.ResponseWriter, r *http.Request) {
	if s.Syslog == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}
	cfg := s.Syslog.Config()
	cfg.CAFile = filepath.Base(cfg.CAFile) // the name, not where it lives
	if cfg.CAFile == "." {
		cfg.CAFile = ""
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"available": true, "registered": s.syslogAllowed(), "admin": s.isAdmin(r),
		"config": cfg, "stats": s.Syslog.Stats(), "formats": syslogout.Formats,
	})
}

func (s *Server) putSyslog(w http.ResponseWriter, r *http.Request) {
	if s.Syslog == nil {
		fail(w, errors.New("syslog is not available here"))
		return
	}
	var in struct {
		Enabled  bool             `json:"enabled"`
		Address  string           `json:"address"`
		Network  string           `json:"network"`
		Format   syslogout.Format `json:"format"`
		MinSev   int              `json:"min_sev"`
		ASCII    bool             `json:"ascii"`
		Insecure bool             `json:"insecure"`
		BaseURL  string           `json:"base_url"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&in); err != nil {
		fail(w, errors.New("bad request"))
		return
	}
	if in.Enabled && !s.allowed(w, license.FeatSyslog) {
		return
	}
	// what the page does not set (facility, ceiling, a CA file given on
	// the command line) stays as it was
	cfg := s.Syslog.Config()
	cfg.Enabled, cfg.Address, cfg.Network, cfg.Format = in.Enabled, clip(in.Address, 255), in.Network, in.Format
	cfg.MinSev, cfg.ASCII, cfg.Insecure, cfg.BaseURL = in.MinSev, in.ASCII, in.Insecure, clip(in.BaseURL, 255)
	if err := cfg.Check(); err != nil {
		fail(w, err)
		return
	}
	if s.DataDir != "" {
		if err := syslogout.SaveConfig(s.syslogFile(), cfg); err != nil {
			fail(w, err)
			return
		}
	}
	s.Syslog.SetConfig(cfg)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) testSyslog(w http.ResponseWriter, r *http.Request) {
	if s.Syslog == nil {
		fail(w, errors.New("syslog is not available here"))
		return
	}
	if !s.allowed(w, license.FeatSyslog) {
		return
	}
	if err := s.Syslog.Test(); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
