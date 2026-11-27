package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/githubflyideas/traffic66/internal/tlscert"
	"github.com/githubflyideas/traffic66/internal/web"
)

func TestUtilityPathsWrongMethodAndMissing(t *testing.T) {
	s, h, _ := apiRulesServer(t)
	s.Static = web.FS()
	for _, p := range []string{"/logo", "/auto", "/tls.crt", "/healthz"} {
		for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			w := apiCall(h, m, p, "", "", "")
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s: %d %s", m, p, w.Code, w.Body)
				continue
			}
			if allow := w.Header().Get("Allow"); allow != http.MethodGet {
				t.Errorf("%s %s Allow=%q", m, p, allow)
			}
			if got := w.Header().Get("Api-Version"); got != apiVersion {
				t.Errorf("%s %s Api-Version=%q", m, p, got)
			}
			if w.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Errorf("%s %s missing nosniff", m, p)
			}
			if w.Header().Get("Referrer-Policy") != "no-referrer" {
				t.Errorf("%s %s missing Referrer-Policy", m, p)
			}
			if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
				t.Errorf("%s %s Content-Type=%q", m, p, w.Header().Get("Content-Type"))
			}
			var body struct{ Error, Code string }
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != "method_not_allowed" || body.Error == "" {
				t.Errorf("%s %s body: %s", m, p, w.Body)
			}
		}
	}

	w := apiCall(h, "GET", "/tls.crt", "", "", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET /tls.crt with no cert: %d %s", w.Code, w.Body)
	}
	if got := w.Header().Get("Api-Version"); got != apiVersion {
		t.Errorf("GET /tls.crt Api-Version=%q", got)
	}
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Errorf("GET /tls.crt Content-Type=%q", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Body.String(), `"code":"not_found"`) {
		t.Errorf("GET /tls.crt body: %s", w.Body)
	}

	// a self-signed cert record whose file is missing is also not_found
	s.Cert = &tlscert.Info{SelfSigned: true, CertPath: "/no/such/traffic66.crt"}
	w = apiCall(h, "GET", "/tls.crt", "", "", "")
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), `"code":"not_found"`) {
		t.Errorf("GET /tls.crt missing file: %d %s", w.Code, w.Body)
	}
}

func TestLogoMissingIsJSONNotFound(t *testing.T) {
	s, h, _ := apiRulesServer(t)
	s.Static = fstest.MapFS{}
	w := apiCall(h, "GET", "/logo", "", "", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET /logo with empty static: %d %s", w.Code, w.Body)
	}
	if got := w.Header().Get("Api-Version"); got != apiVersion {
		t.Errorf("Api-Version=%q", got)
	}
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") || !strings.Contains(w.Body.String(), `"code":"not_found"`) {
		t.Errorf("body: %s type %s", w.Body, w.Header().Get("Content-Type"))
	}
}
