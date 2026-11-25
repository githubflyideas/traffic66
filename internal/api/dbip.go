package api

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/geo"
)

// dbipBase is where DB-IP publishes its free Lite databases, one file per
// month: dbip-country-lite-2026-10.mmdb.gz.
var dbipBase = "https://download.db-ip.com/free/"

// fetchDBIP downloads this month's (or, early in a month, last month's)
// DB-IP Lite database of one kind ("country" or "asn").
func fetchDBIP(ctx context.Context, kind string, now time.Time) (*geo.Reader, []byte, error) {
	var last error
	for _, m := range []time.Time{now, now.AddDate(0, 0, -now.Day())} {
		url := fmt.Sprintf("%sdbip-%s-lite-%s.mmdb.gz", dbipBase, kind, m.Format("2006-01"))
		b, err := download(ctx, url)
		if err != nil {
			last = err
			continue
		}
		r, err := geo.FromBytes(b)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", url, err)
		}
		if k := r.Kind(); k != kind && k != "both" {
			return nil, nil, fmt.Errorf("%s: holds %s, not %s", url, k, kind)
		}
		return r, b, nil
	}
	return nil, nil, last
}

func download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "traffic66")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	zr, err := gzip.NewReader(io.LimitReader(resp.Body, maxGeoUpload))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", url, err)
	}
	b, err := io.ReadAll(io.LimitReader(zr, maxGeoUpload))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", url, err)
	}
	return b, nil
}

// updateDBIP downloads the latest DB-IP Lite country and ASN databases and
// uses them instead of the built-in ones.
func (s *Server) updateDBIP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	now := time.Now().UTC()
	cty, cb, err := fetchDBIP(ctx, "country", now)
	if err != nil {
		fail(w, fmt.Errorf("download: %w", err))
		return
	}
	as, ab, err := fetchDBIP(ctx, "asn", now)
	if err != nil {
		fail(w, fmt.Errorf("download: %w", err))
		return
	}
	for name, b := range map[string][]byte{enrich.DBIPCountryFile: cb, enrich.DBIPASNFile: ab} {
		if err := writeFileAtomic(filepath.Join(s.DataDir, name), b); err != nil {
			fail(w, err)
			return
		}
	}
	s.ASN.SetFallback(cty, as, enrich.DBIPCountryFile, enrich.DBIPASNFile)
	log.Printf("geo: DB-IP Lite downloaded (country %s, ASN %s)", cty.Built.Format("2006-01-02"), as.Built.Format("2006-01-02"))
	writeJSON(w, http.StatusOK, map[string]any{"sources": s.ASN.Sources()})
}

// deleteGeo removes an installed database; what it held falls back to the
// remaining ones and the built-in DB-IP Lite.
func (s *Server) deleteGeo(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("file")
	dir := s.DataDir
	switch {
	case slices.Contains(enrich.MMDBFiles, name):
		os.Remove(filepath.Join(dir, name))
		s.ASN.LoadMMDBs(dir)
	case name == enrich.DBIPCountryFile || name == enrich.DBIPASNFile:
		os.Remove(filepath.Join(dir, name))
		bc, ba, err := geo.Builtin()
		if err != nil {
			fail(w, err)
			return
		}
		if name == enrich.DBIPCountryFile {
			s.ASN.SetFallback(bc, nil, "built-in", "")
		} else {
			s.ASN.SetFallback(nil, ba, "", "built-in")
		}
	case name != "" && name == s.ASN.TableFile():
		os.Remove(filepath.Join(dir, name))
		s.ASN.TakeTable(enrich.NewASNDB(), "")
	default:
		fail(w, errors.New("no such country or ASN file"))
		return
	}
	log.Printf("geo: removed %s", name)
	writeJSON(w, http.StatusOK, map[string]any{"sources": s.ASN.Sources()})
}
