package api

import (
	"bytes"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubflyideas/traffic66/internal/web"
)

func TestLogo(t *testing.T) {
	dir := t.TempDir()
	s := &Server{DataDir: dir, Static: web.FS()}
	get := func() (string, string) {
		w := httptest.NewRecorder()
		s.logo(w, httptest.NewRequest("GET", "/logo", nil))
		return w.Header().Get("Content-Type"), w.Body.String()
	}
	if ct, body := get(); ct != "image/svg+xml" || !strings.Contains(body, "traffic") {
		t.Fatalf("default logo: %s %.60s", ct, body)
	}
	put := func(b []byte) int {
		w := httptest.NewRecorder()
		s.putLogo(w, httptest.NewRequest("POST", "/api/logo", bytes.NewReader(b)))
		return w.Code
	}
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 40)...)
	if c := put(png); c != 200 {
		t.Fatal("png rejected", c)
	}
	if ct, body := get(); ct != "image/png" || body != string(png) {
		t.Fatalf("custom logo not served: %s", ct)
	}
	if c := put([]byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`)); c != 200 {
		t.Fatal("svg rejected", c)
	}
	if _, err := os.Stat(filepath.Join(dir, "logo.png")); err == nil {
		t.Fatal("old logo kept beside the new one")
	}
	if c := put([]byte("hello")); c != 400 {
		t.Fatal("text accepted as a logo", c)
	}
	if c := put(make([]byte, maxLogo+1)); c != 400 {
		t.Fatal("oversized logo accepted", c)
	}
	w := httptest.NewRecorder()
	s.deleteLogo(w, httptest.NewRequest("DELETE", "/api/logo", nil))
	if ct, _ := get(); ct != "image/svg+xml" {
		t.Fatal("reset did not restore the built-in logo")
	}
	w = httptest.NewRecorder()
	s.logo(w, httptest.NewRequest("GET", "/logo", nil))
	if !strings.Contains(w.Header().Get("Content-Security-Policy"), "sandbox") {
		t.Fatal("logo served without sandbox")
	}
}
