package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/githubflyideas/traffic66/internal/license"
)

// putLicense installs a licence from the settings page. Until now the only
// way to register was to put license.json into the data directory by hand,
// which means a shell on the machine and knowing where the data directory
// is — a strange thing to require of the one step that turns a paying
// customer into a registered installation.
//
// Nothing is written until the licence has been checked the same way the
// checker checks it at startup: a file that is unsigned, signed by a key
// this release does not accept, issued for another installation, or
// expired is refused with the reason, rather than written and discovered
// to be useless at the next reload.
func (s *Server) putLicense(w http.ResponseWriter, r *http.Request) {
	if s.License == nil {
		fail(w, errors.New("no licence state on this installation"))
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&in); err != nil {
		fail(w, errors.New("bad request"))
		return
	}
	f, err := license.Parse([]byte(in.Text))
	if err != nil {
		fail(w, err)
		return
	}
	now := time.Now()
	if st := license.Check(s.License.Ring, f, s.License.Install, now); st.Kind != license.KindLicensed {
		fail(w, errors.New(st.Reason))
		return
	}
	if err := license.Save(s.License.Path, f); err != nil {
		fail(w, err)
		return
	}
	// Take effect now rather than at the next few-hourly check: somebody is
	// watching the page they just installed it on.
	writeJSON(w, http.StatusOK, map[string]any{"license": s.License.Check(now).Public()})
}
