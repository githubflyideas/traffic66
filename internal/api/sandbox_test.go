package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/sandbox"
	"github.com/githubflyideas/traffic66/internal/store"
)

// ds=sb answers from the sandbox; without it the live database answers and
// has none of the capture's data.
func TestSandboxData(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(store.Options{Dir: filepath.Join(dir, "live")})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	sb := sandbox.New(filepath.Join(dir, "sandbox"), enrich.NewInventory(), enrich.NewASNDB(), enrich.NewThreats())
	defer sb.Close()
	s := &Server{Store: st, SB: sb, Inv: enrich.NewInventory(), ASN: enrich.NewASNDB(), Thr: enrich.NewThreats()}
	h := s.data((*Server).findings)
	end := time.Now().Add(-3 * time.Hour).Truncate(time.Minute)
	q := fmt.Sprintf("from=%d&to=%d", end.Add(-time.Hour).UnixMilli(), end.Add(time.Hour).UnixMilli())
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest("GET", "/api/findings?ds=sb&"+q, nil))
	if w.Code != 400 {
		t.Fatalf("empty sandbox: %d %s", w.Code, w.Body)
	}
	var buf bytes.Buffer
	sandbox.Sample(&buf, end)
	w = httptest.NewRecorder()
	s.putSandboxFile(w, httptest.NewRequest("POST", "/api/sandbox/files?name=a.pcap", &buf))
	if w.Code != 200 {
		t.Fatalf("upload: %d %s", w.Code, w.Body)
	}
	for i := 0; i < 300 && !sb.Info().Ready; i++ {
		time.Sleep(100 * time.Millisecond)
	}
	for sb.Info().Busy {
		time.Sleep(100 * time.Millisecond)
	}
	count := func(ds string) int {
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest("GET", "/api/findings?"+ds+q, nil))
		var out struct{ Findings []any }
		json.Unmarshal(w.Body.Bytes(), &out)
		return len(out.Findings)
	}
	if n := count("ds=sb&"); n != 3 {
		t.Errorf("sandbox findings: %d", n)
	}
	if n := count(""); n != 0 {
		t.Errorf("live findings: %d", n)
	}
	w = httptest.NewRecorder()
	s.putSandboxFile(w, httptest.NewRequest("POST", "/api/sandbox/files?name=b.txt", bytes.NewReader([]byte("hello there"))))
	if w.Code != 415 {
		t.Errorf("text file: %d", w.Code)
	}
}
