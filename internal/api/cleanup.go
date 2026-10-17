package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/githubflyideas/traffic66/internal/license"
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

// allowed reports whether a registered-only feature may be used, and
// answers the request when it may not. An installation with no licence
// checker at all (the offline sandbox) is treated as unregistered: the
// answer is never more permissive than a missing licence.
func (s *Server) allowed(w http.ResponseWriter, feature string) bool {
	if s.License != nil && s.License.State().Allowed(feature) {
		return true
	}
	writeJSON(w, http.StatusForbidden, map[string]string{"error": "not_registered", "feature": feature})
	return false
}

// ifaceTarget reads the interface and age a per-interface cleanup names.
// days 0 means the interface's whole history.
func ifaceTarget(exporter, ifindex, days string) (string, uint32, time.Time, error) {
	var before time.Time
	if _, err := netip.ParseAddr(exporter); err != nil {
		return "", 0, before, errors.New("exporter must be an address")
	}
	idx, err := strconv.ParseUint(ifindex, 10, 32)
	if err != nil || idx == 0 {
		return "", 0, before, errors.New("ifindex missing")
	}
	if days != "" {
		d, err := strconv.Atoi(days)
		if err != nil || d < 0 {
			return "", 0, before, errors.New("days must be 0 or more")
		}
		if d > 0 {
			before = time.Now().AddDate(0, 0, -d)
		}
	}
	return exporter, uint32(idx), before, nil
}

// getCleanupIface shows what deleting one interface's data would remove.
func (s *Server) getCleanupIface(w http.ResponseWriter, r *http.Request) {
	if !s.allowed(w, license.FeatPurgeIface) {
		return
	}
	q := r.URL.Query()
	exp, idx, before, err := ifaceTarget(q.Get("exporter"), q.Get("ifindex"), q.Get("days"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.Store.PurgeSizeIface(exp, idx, before))
}

// postCleanupIface deletes one interface's flow records and counters.
func (s *Server) postCleanupIface(w http.ResponseWriter, r *http.Request) {
	if !s.allowed(w, license.FeatPurgeIface) {
		return
	}
	var in struct {
		Exporter string `json:"exporter"`
		IfIndex  uint32 `json:"ifindex"`
		Days     int    `json:"days"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&in); err != nil {
		fail(w, errors.New("bad request"))
		return
	}
	exp, idx, before, err := ifaceTarget(in.Exporter, strconv.FormatUint(uint64(in.IfIndex), 10), strconv.Itoa(in.Days))
	if err != nil {
		fail(w, err)
		return
	}
	disk := s.Store.Usage().DiskBytes
	done, err := s.Store.PurgeIface(exp, idx, before)
	if err != nil {
		fail(w, err)
		return
	}
	// what the files lost, plus what the database file gave back
	done.Bytes += max(0, disk-s.Store.Usage().DiskBytes)
	writeJSON(w, http.StatusOK, done)
}
