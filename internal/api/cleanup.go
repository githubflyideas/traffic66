package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"net/netip"
	"strings"
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
	exps, _ := s.Store.StoredExporters()
	writeJSON(w, http.StatusOK, map[string]any{"options": opts, "disk_bytes": u.DiskBytes, "oldest": u.Oldest, "exporters": exps})
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
	if s.Pipe != nil {
		s.Pipe.Forget(before)
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

// allowed reports whether a registered-only feature may be used, and
// answers the request when it may not. An installation with no licence
// checker at all (the offline sandbox) is treated as unregistered: the
// answer is never more permissive than a missing licence.
//
// The registered build's features call it; this build has none that do.
func (s *Server) allowed(w http.ResponseWriter, feature string) bool {
	if s.License != nil && s.License.State().Allowed(feature) {
		return true
	}
	writeJSON(w, http.StatusForbidden, map[string]string{"error": "not_registered", "feature": feature})
	return false
}

// postPurgeExporter deletes all flow records of one device, for a device
// whose data should not be there (forged or broken exports). It asks for
// the device's address and an explicit confirmation.
func (s *Server) postPurgeExporter(w http.ResponseWriter, r *http.Request) {
	typ, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if typ != "application/json" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "application/json required"})
		return
	}
	var in struct {
		Exporter string `json:"exporter"`
		Confirm  bool   `json:"confirm"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&in); err != nil || !in.Confirm {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "exporter and confirm=true required"})
		return
	}
	a, err := netip.ParseAddr(strings.TrimSpace(in.Exporter))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_exporter"})
		return
	}
	disk := s.Store.Usage().DiskBytes
	var done store.PurgeSize
	if s.Pipe != nil {
		done, err = s.Pipe.PurgeExporter(a.Unmap().String())
	} else {
		done, err = s.Store.PurgeExporter(a.Unmap().String())
	}
	if err != nil {
		fail(w, err)
		return
	}
	done.Bytes = max(0, disk-s.Store.Usage().DiskBytes)
	log.Printf("cleanup: deleted %d flow records of device %s", done.Rows, a.Unmap())
	writeJSON(w, http.StatusOK, done)
}
