package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/githubflyideas/traffic66/internal/store"
)

// findings lists what the detection rules found in the time range, with
// counts of open findings by severity and the detector's state.
func (s *Server) findings(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
	if err != nil {
		fail(w, err)
		return
	}
	v := r.URL.Query()
	status := v.Get("status")
	if status == "" {
		status = store.FindingOpen
	}
	fs, err := s.Store.Findings(store.FindingQuery{From: q.From, To: q.To, Status: status, IP: v.Get("ip"), Limit: limitParam(r, 500)})
	if err != nil {
		fail(w, err)
		return
	}
	open, err := s.Store.Findings(store.FindingQuery{From: q.From, To: q.To, Status: store.FindingOpen, IP: v.Get("ip"), Limit: 1000})
	if err != nil {
		fail(w, err)
		return
	}
	counts := map[int]int{1: 0, 2: 0, 3: 0}
	for _, f := range open {
		counts[f.Sev]++
	}
	out := map[string]any{"findings": fs, "open": map[string]int{"high": counts[3], "medium": counts[2], "low": counts[1]}}
	if s.Det != nil {
		out["detector"] = s.Det.Status()
	}
	writeJSON(w, http.StatusOK, out)
}

// setFindings marks findings as dealt with, as not a problem, or open again.
func (s *Server) setFindings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs    []int64 `json:"ids"`
		Status string  `json:"status"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil {
		fail(w, err)
		return
	}
	if len(in.IDs) == 0 {
		fail(w, errors.New("no findings given"))
		return
	}
	if err := s.Store.SetFindingStatus(in.IDs, in.Status); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
