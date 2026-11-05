package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/githubflyideas/traffic66/internal/enrich"
)

// Threat lists can be imported from the threat page as well as dropped
// into <data>/threats by hand: a file or a paste, one address or network
// per line, kept as <data>/threats/<name>.txt and in use at once, without
// a restart. Flows that already arrived keep what they were matched
// against; a list applies to what comes in after it.
//
// Nothing is fetched from anywhere: the operator brings the list. That
// keeps the installation from reaching out to the internet on a schedule,
// as it does for nothing else.

var threatName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,39}$`)

const maxThreatList = 32 << 20

func (s *Server) threatDir() string { return filepath.Join(s.DataDir, "threats") }

// putThreatList imports or replaces one list.
func (s *Server) putThreatList(w http.ResponseWriter, r *http.Request) {
	if s.Thr == nil || s.DataDir == "" {
		fail(w, errors.New("threat lists cannot be saved on this installation"))
		return
	}
	var in struct {
		Name string `json:"name"`
		Text string `json:"text"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxThreatList+4096)).Decode(&in); err != nil {
		fail(w, errors.New("the list is too large or not readable"))
		return
	}
	in.Name = strings.TrimSuffix(strings.TrimSpace(in.Name), ".txt")
	if !threatName.MatchString(in.Name) {
		fail(w, errors.New("a list name is letters, digits, dot, dash or underscore, up to 40"))
		return
	}
	// read it on its own first: a file with no address in it is refused
	// rather than replacing a working list with an empty one
	n, err := enrich.NewThreats().Add(in.Name, strings.NewReader(in.Text))
	if err != nil {
		fail(w, err)
		return
	}
	if n == 0 {
		fail(w, errors.New("no address or network found in the list"))
		return
	}
	if err := os.MkdirAll(s.threatDir(), 0o755); err != nil {
		fail(w, err)
		return
	}
	path := filepath.Join(s.threatDir(), in.Name+".txt")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(in.Text), 0o644); err != nil {
		fail(w, err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		fail(w, err)
		return
	}
	if _, err := s.Thr.Add(in.Name, strings.NewReader(in.Text)); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": in.Name, "entries": n, "lists": s.Thr.Lists()})
}

// deleteThreatList stops using one list and deletes its file.
func (s *Server) deleteThreatList(w http.ResponseWriter, r *http.Request) {
	if s.Thr == nil || s.DataDir == "" {
		fail(w, errors.New("threat lists cannot be changed on this installation"))
		return
	}
	name := r.URL.Query().Get("name")
	if !threatName.MatchString(name) {
		fail(w, errors.New("bad list name"))
		return
	}
	if _, ok := s.Thr.Lists()[name]; !ok {
		fail(w, fmt.Errorf("no list called %s", name))
		return
	}
	if err := os.Remove(filepath.Join(s.threatDir(), name+".txt")); err != nil && !os.IsNotExist(err) {
		fail(w, err)
		return
	}
	s.Thr.Remove(name)
	writeJSON(w, http.StatusOK, map[string]any{"lists": s.Thr.Lists()})
}
