package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/githubflyideas/traffic66/internal/enrich"
)

func TestGeoUpload(t *testing.T) {
	dir := t.TempDir()
	s := &Server{ASN: enrich.NewASNDB(), DataDir: dir}
	up := func(b []byte) (int, map[string]any) {
		w := httptest.NewRecorder()
		s.putGeo(w, httptest.NewRequest("POST", "/api/geo", bytes.NewReader(b)))
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	for _, f := range []string{"country.mmdb", "asn.mmdb"} {
		b, err := os.ReadFile(filepath.Join("..", "geo", "testdata", f))
		if err != nil {
			t.Fatal(err)
		}
		if code, out := up(b); code != 200 || out["installed"] != f {
			t.Fatalf("%s: %d %v", f, code, out)
		}
	}
	tsv := "198.51.100.0\t198.51.100.255\t64501\tNL\tExample Table Net\n"
	if code, out := up([]byte(tsv)); code != 200 || out["installed"] != "asn.tsv.gz" {
		t.Fatalf("tsv: %d %v", code, out)
	}
	if code, _ := up([]byte("hello world\n")); code != 400 {
		t.Fatalf("garbage accepted: %d", code)
	}
	for _, tc := range []struct {
		ip      string
		asn     uint32
		cc, org string
	}{
		{"8.8.8.8", 15169, "US", "Google LLC"},             // country and ASN from the two .mmdb files
		{"198.51.100.7", 64501, "NL", "Example Table Net"}, // only in the table
		{"1.0.0.1", 0, "AU", ""},                           // only a country
	} {
		asn, cc, org := s.ASN.Lookup(netip.MustParseAddr(tc.ip))
		if asn != tc.asn || cc != tc.cc || org != tc.org {
			t.Errorf("%s: %d %s %s", tc.ip, asn, cc, org)
		}
	}
	for _, f := range []string{"country.mmdb", "asn.mmdb", "asn.tsv.gz"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s not saved: %v", f, err)
		}
	}
	if n := len(s.ASN.Sources()); n != 3 {
		t.Errorf("%d sources", n)
	}
}
