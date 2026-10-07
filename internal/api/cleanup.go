package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/githubflyideas/traffic66/internal/store"
)

// cleanupDays are the ages offered for deleting old data.
var cleanupDays = []int{120, 90, 60, 30, 7}

// getCleanup shows what each cleanup would remove.
func (s *Server) getCleanup(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	type opt struct {
		Days int `json:"days"` // 0: all
		store.PurgeSize
	}
	var opts []opt
	for _, d := range cleanupDays {
		opts = append(opts, opt{d, s.Store.PurgeSizeBefore(now.AddDate(0, 0, -d))})
	}
	all := s.Store.PurgeSizeBefore(time.Time{})
	opts = append(opts, opt{0, all})
	u := s.Store.Usage()
	// the database file shrinks too: count its share by records
	for i := range opts {
		if all.Rows > 0 {
			opts[i].Bytes = max(opts[i].Bytes, u.DiskBytes*opts[i].Rows/all.Rows)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"options": opts, "disk_bytes": u.DiskBytes, "oldest": u.Oldest})
}

// postCleanup deletes the data older than the given days (0: all data).
func (s *Server) postCleanup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Days *int `json:"days"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&in); err != nil || in.Days == nil {
		fail(w, errors.New("days missing"))
		return
	}
	var before time.Time
	switch {
	case *in.Days == 0:
	case *in.Days > 0:
		before = time.Now().AddDate(0, 0, -*in.Days)
	default:
		fail(w, errors.New("days must be 0 or more"))
		return
	}
	disk := s.Store.Usage().DiskBytes
	done, err := s.Store.PurgeBefore(before)
	if err != nil {
		fail(w, err)
		return
	}
	// what was freed on disk, including files rewritten without old records
	done.Bytes = max(0, disk-s.Store.Usage().DiskBytes)
	writeJSON(w, http.StatusOK, done)
}
