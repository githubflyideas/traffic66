package api

import (
	"errors"
	"net/http"

	"github.com/githubflyideas/traffic66/internal/sandbox"
)

// data serves a page's data from the live database, or with ds=sb from the
// sandbox, where uploaded capture files are analysed apart from the live
// data.
func (s *Server) data(h func(*Server, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ds") != "sb" {
			h(s, w, r)
			return
		}
		if s.SB == nil {
			fail(w, errors.New("offline analysis is not available"))
			return
		}
		st, det, inv := s.SB.Store()
		if st == nil {
			fail(w, errors.New("no capture files imported yet"))
			return
		}
		c := &Server{Store: st, Det: det, Inv: inv, ASN: s.ASN, Thr: s.Thr, DNS: s.DNS, Static: s.Static, Version: s.Version,
			Demo: s.Demo, DataDir: s.DataDir, Pipe: s.Pipe, Col: s.Col, SB: s.SB}
		h(c, w, r)
	}
}

func (s *Server) getSandbox(w http.ResponseWriter, r *http.Request) {
	if s.SB == nil {
		fail(w, errors.New("offline analysis is not available"))
		return
	}
	writeJSON(w, http.StatusOK, s.SB.Info())
}

// putSandboxFile receives one capture file (the request body) and queues it
// for import. Size and count limits are checked while it streams to disk.
func (s *Server) putSandboxFile(w http.ResponseWriter, r *http.Request) {
	if s.SB == nil {
		fail(w, errors.New("offline analysis is not available"))
		return
	}
	if r.ContentLength > s.SB.Lim.FileSize {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"error": "too_big", "max": s.SB.Lim.FileSize})
		return
	}
	f, err := s.SB.Add(r.URL.Query().Get("name"), r.Body, false)
	if errors.Is(err, sandbox.ErrLimit) {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"error": err.Error(), "max": s.SB.Lim.FileSize, "max_files": s.SB.Lim.Files})
		return
	}
	if errors.Is(err, sandbox.ErrNotCapture) {
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]any{"error": err.Error(), "code": "not_capture"})
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"file": f, "sandbox": s.SB.Info()})
}

// deleteSandbox deletes one file (?name=) or all of them, and the data.
func (s *Server) deleteSandbox(w http.ResponseWriter, r *http.Request) {
	if s.SB == nil {
		fail(w, errors.New("offline analysis is not available"))
		return
	}
	if err := s.SB.Delete(r.URL.Query().Get("name")); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.SB.Info())
}

// putSandboxActive makes one capture file the one the pages show.
func (s *Server) putSandboxActive(w http.ResponseWriter, r *http.Request) {
	if s.SB == nil {
		fail(w, errors.New("offline analysis is not available"))
		return
	}
	if err := s.SB.Select(r.URL.Query().Get("name")); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.SB.Info())
}
