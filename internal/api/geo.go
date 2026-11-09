package api

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/geo"
)

// maxGeoUpload bounds an uploaded database (GeoLite2-City is about 70 MB).
const maxGeoUpload = 512 << 20

func (s *Server) getGeo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"sources": s.ASN.Sources()})
}

// putGeo installs an uploaded country/ASN database: a MaxMind DB file
// (.mmdb, country or ASN) or a tab-separated IP-to-ASN table (plain or
// gzip). It is checked, saved in the data directory, and used for new flows
// at once; flows already stored keep the country and network they got.
func (s *Server) putGeo(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxGeoUpload))
	if err != nil {
		fail(w, fmt.Errorf("upload: %w", err))
		return
	}
	if len(b) == 0 {
		fail(w, errors.New("empty file"))
		return
	}
	var name string
	switch {
	case geo.IsMMDB(b):
		rd, err := geo.FromBytes(b)
		if err != nil {
			fail(w, err)
			return
		}
		name = rd.Kind() + ".mmdb"
		if err := writeFileAtomic(filepath.Join(s.DataDir, name), b); err != nil {
			fail(w, err)
			return
		}
		if name == "both.mmdb" { // replaces the separate ones
			os.Remove(filepath.Join(s.DataDir, "country.mmdb"))
			os.Remove(filepath.Join(s.DataDir, "asn.mmdb"))
		}
		for _, err := range s.ASN.LoadMMDBs(s.DataDir) {
			log.Printf("geo: %v", err)
		}
	default:
		plain := b
		if len(b) > 2 && b[0] == 0x1f && b[1] == 0x8b {
			zr, err := gzip.NewReader(bytes.NewReader(b))
			if err != nil {
				fail(w, err)
				return
			}
			if plain, err = io.ReadAll(io.LimitReader(zr, 4*maxGeoUpload)); err != nil {
				fail(w, fmt.Errorf("gzip: %w", err))
				return
			}
		}
		tbl, err := enrich.LoadTable(bytes.NewReader(plain))
		if err != nil {
			fail(w, fmt.Errorf("not a MaxMind DB (.mmdb) and not an IP-to-ASN table: %w", err))
			return
		}
		gz := b
		if &plain[0] == &b[0] { // uploaded uncompressed: store compressed
			var buf bytes.Buffer
			zw := gzip.NewWriter(&buf)
			zw.Write(plain)
			zw.Close()
			gz = buf.Bytes()
		}
		name = "asn.tsv.gz"
		if err := writeFileAtomic(filepath.Join(s.DataDir, name), gz); err != nil {
			fail(w, err)
			return
		}
		// older tables would be preferred at the next start; keep them aside
		for _, old := range []string{"asn.tsv", "ip2asn-combined.tsv.gz"} {
			p := filepath.Join(s.DataDir, old)
			if _, err := os.Stat(p); err == nil {
				os.Rename(p, p+".replaced")
			}
		}
		s.ASN.TakeTable(tbl, name)
	}
	log.Printf("geo: installed %s (%d bytes) from upload", name, len(b))
	writeJSON(w, http.StatusOK, map[string]any{"installed": name, "sources": s.ASN.Sources()})
}

func writeFileAtomic(path string, b []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
