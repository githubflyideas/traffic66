package api

import (
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/githubflyideas/traffic66/internal/pipeline"
)

// /metrics needs a sign-in (Basic auth for a scraper) and speaks the
// Prometheus text format.
func TestMetrics(t *testing.T) {
	s := &Server{Users: map[string]string{"admin": "right"}, Version: "9.9.9", Pipe: &pipeline.Pipeline{}, Static: fstest.MapFS{}}
	s.Pipe.Records.Add(42)
	s.Pipe.Implausible.Add(3)
	h := s.Handler()
	r := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("without a sign-in: %d, want 401", w.Code)
	}
	r.SetBasicAuth("admin", "right")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	body := w.Body.String()
	if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain; version=0.0.4") {
		t.Fatalf("%d %s", w.Code, w.Header().Get("Content-Type"))
	}
	for _, want := range []string{`traffic66_build_info{version="9.9.9"`, "# TYPE traffic66_records_total counter", "traffic66_records_total 42", "traffic66_records_refused_total 3"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in\n%s", want, body)
		}
	}
}
