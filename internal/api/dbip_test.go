package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubflyideas/traffic66/internal/enrich"
)

func lookup(s *Server, ip string) (uint32, string, string) {
	return s.ASN.Lookup(netip.MustParseAddr(ip))
}

// The built-in DB-IP Lite answers only what the operator's databases do not;
// removing an operator database falls back to it.
func TestGeoFallback(t *testing.T) {
	dir := t.TempDir()
	s := &Server{ASN: enrich.NewASNDB(), DataDir: dir}
	if err := s.ASN.LoadDBIP(dir); err != nil {
		t.Fatal(err)
	}
	if asn, cc, _ := lookup(s, "1.1.1.1"); asn != 13335 || cc == "" {
		t.Fatalf("built-in: %d %q", asn, cc)
	}
	b, _ := os.ReadFile(filepath.Join("..", "geo", "testdata", "ipinfo.mmdb"))
	w := httptest.NewRecorder()
	s.putGeo(w, httptest.NewRequest("POST", "/api/geo", bytes.NewReader(b)))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"installed":"both.mmdb"`) {
		t.Fatalf("upload: %d %s", w.Code, w.Body)
	}
	if asn, cc, org := lookup(s, "1.1.1.1"); asn != 13335 || cc != "AU" || org != "Cloudflare, Inc." {
		t.Errorf("uploaded: %d %s %s", asn, cc, org)
	}
	if asn, cc, _ := lookup(s, "9.9.9.9"); asn == 0 || cc == "" { // not in the upload
		t.Errorf("fallback: %d %q", asn, cc)
	}
	kinds := ""
	for _, src := range s.ASN.Sources() {
		kinds += src.Kind + ":" + src.File + " "
	}
	if kinds != "both:both.mmdb country:built-in asn:built-in " {
		t.Errorf("sources %q", kinds)
	}
	w = httptest.NewRecorder()
	s.deleteGeo(w, httptest.NewRequest("DELETE", "/api/geo?file=both.mmdb", nil))
	if w.Code != 200 {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	if _, err := os.Stat(filepath.Join(dir, "both.mmdb")); err == nil {
		t.Error("file kept")
	}
	if n := len(s.ASN.Sources()); n != 2 {
		t.Errorf("%d sources after delete", n)
	}
	w = httptest.NewRecorder()
	s.deleteGeo(w, httptest.NewRequest("DELETE", "/api/geo?file=../inventory.txt", nil))
	if w.Code != 400 {
		t.Errorf("unknown file: %d", w.Code)
	}
}

// Update downloads this month's or last month's DB-IP Lite files.
func TestDBIPUpdate(t *testing.T) {
	gz := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join("..", "geo", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		zw.Write(b)
		zw.Close()
		return buf.Bytes()
	}
	files := map[string][]byte{}
	var asked []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.Path)
		for k, b := range files {
			if strings.HasPrefix(r.URL.Path, "/free/"+k) {
				w.Write(b)
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()
	old := dbipBase
	dbipBase = ts.URL + "/free/"
	defer func() { dbipBase = old }()

	dir := t.TempDir()
	s := &Server{ASN: enrich.NewASNDB(), DataDir: dir}
	s.ASN.LoadDBIP(dir)
	post := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.updateDBIP(w, httptest.NewRequest("POST", "/api/geo/dbip", nil))
		return w
	}
	if w := post(); w.Code != 400 {
		t.Fatalf("nothing to download: %d", w.Code)
	}
	files["dbip-country-lite-"] = gz("country.mmdb")
	files["dbip-asn-lite-"] = gz("asn.mmdb")
	w := post()
	if w.Code != 200 {
		t.Fatalf("update: %d %s", w.Code, w.Body)
	}
	var out struct{ Sources []enrich.GeoSource }
	json.Unmarshal(w.Body.Bytes(), &out)
	if len(out.Sources) != 2 || out.Sources[0].File != enrich.DBIPCountryFile || out.Sources[1].File != enrich.DBIPASNFile || !out.Sources[0].Fallback {
		t.Errorf("sources %+v", out.Sources)
	}
	for _, f := range []string{enrich.DBIPCountryFile, enrich.DBIPASNFile} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Error(err)
		}
	}
	if asn, cc, _ := lookup(s, "8.8.8.8"); asn != 15169 || cc != "US" {
		t.Errorf("after update: %d %s", asn, cc)
	}
	// both months were tried before the files existed
	if len(asked) < 2 || asked[0] == asked[1] {
		t.Errorf("asked %v", asked)
	}
}
