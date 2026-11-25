package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManual(t *testing.T) {
	s := &Server{}
	get := func(url string, h http.HandlerFunc, name string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", url, nil)
		if name != "" {
			r.SetPathValue("name", name)
		}
		w := httptest.NewRecorder()
		h(w, r)
		return w
	}
	if w := get("/api/manual?lang=ja", s.manual, ""); w.Code != 200 || !strings.Contains(w.Body.String(), "**日本語**") {
		t.Errorf("ja: %d", w.Code)
	}
	for _, l := range []string{"xx", "../x", "", "en"} {
		if w := get("/api/manual?lang="+l, s.manual, ""); w.Code != 200 || !strings.Contains(w.Body.String(), "**English**") {
			t.Errorf("%q does not fall back to English: %d", l, w.Code)
		}
	}
	if w := get("/", s.manualImage, "overview.png"); w.Code != 200 || w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("image: %d", w.Code)
	}
	for _, n := range []string{"../MANUAL.md", "MANUAL.md", "x.png"} {
		if w := get("/", s.manualImage, n); w.Code != 404 {
			t.Errorf("%q: %d", n, w.Code)
		}
	}
}
