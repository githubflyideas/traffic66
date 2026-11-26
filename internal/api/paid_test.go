package api

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// The registered pages — interface check, findings and the inventory,
// beside the traffic statistics — answer not_registered on an
// unregistered installation, reads and writes alike, and answer on a
// registered one and in the demo. The free pages answer everywhere.
func TestPaidPages(t *testing.T) {
	paid := []struct{ method, path, body string }{
		{"GET", "/api/recon?range=24h&exporter=10.0.0.1&ifindex=1", ""},
		{"GET", "/api/findings?range=24h", ""},
		{"POST", "/api/findings", `{"id":"x","status":"done"}`},
		{"GET", "/api/inventory", ""},
		{"POST", "/api/inventory", `{"text":"host 10.0.0.5 PC\n"}`},
		{"POST", "/api/iface", `{"exporter":"10.0.0.1","ifindex":1,"name":"WAN"}`},
	}
	run := func(s *Server) func(method, path, body string) *httptest.ResponseRecorder {
		h := s.Handler()
		return func(method, path, body string) *httptest.ResponseRecorder {
			r := httptest.NewRequest(method, path, strings.NewReader(body))
			r.SetBasicAuth("admin", "pw")
			if body != "" {
				r.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			return w
		}
	}
	for _, mode := range []string{"unregistered", "registered", "demo"} {
		s, _ := billingServer(t)
		wire(s)
		switch mode {
		case "unregistered":
			s.License = nil
		case "demo":
			s.License, s.Demo = nil, true
		}
		call := run(s)
		for _, p := range paid {
			w := call(p.method, p.path, p.body)
			refused := w.Code == 403 && strings.Contains(w.Body.String(), `"not_registered"`)
			if refused != (mode == "unregistered") {
				t.Errorf("%s %s %s: %d %.120s", mode, p.method, p.path, w.Code, w.Body)
			}
		}
		for _, p := range []string{"/api/overview?range=24h", "/api/topn?range=24h&dim=client&by=wire", "/api/records?range=1h"} {
			if w := call("GET", p, ""); w.Code != 200 {
				t.Errorf("%s free page %s: %d", mode, p, w.Code)
			}
		}
	}
}
