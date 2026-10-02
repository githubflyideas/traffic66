package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// maxLogo bounds an uploaded logo.
const maxLogo = 1 << 20

var logoTypes = map[string]string{"svg": "image/svg+xml", "png": "image/png", "jpg": "image/jpeg", "webp": "image/webp", "gif": "image/gif"}

// logoKind tells an image's type from its first bytes.
func logoKind(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return "png"
	case bytes.HasPrefix(b, []byte{0xff, 0xd8, 0xff}):
		return "jpg"
	case len(b) > 12 && bytes.HasPrefix(b, []byte("RIFF")) && string(b[8:12]) == "WEBP":
		return "webp"
	case bytes.HasPrefix(b, []byte("GIF8")):
		return "gif"
	}
	head := bytes.ToLower(b[:min(len(b), 1024)])
	if bytes.Contains(head, []byte("<svg")) {
		return "svg"
	}
	return ""
}

// customLogo returns the path and type of the uploaded logo, if any.
func (s *Server) customLogo() (string, string) {
	if s.DataDir == "" {
		return "", ""
	}
	for ext := range logoTypes {
		p := filepath.Join(s.DataDir, "logo."+ext)
		if _, err := os.Stat(p); err == nil {
			return p, ext
		}
	}
	return "", ""
}

// logo serves the uploaded logo, or the built-in one. It needs no sign-in:
// the sign-in page shows it. An uploaded SVG is served so that a browser
// opening it directly runs no script in it.
func (s *Server) logo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:; sandbox")
	if p, ext := s.customLogo(); p != "" {
		b, err := os.ReadFile(p)
		if err == nil {
			w.Header().Set("Content-Type", logoTypes[ext])
			w.Write(b)
			return
		}
	}
	b, err := fs.ReadFile(s.Static, "logo.svg")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write(b)
}

// putLogo replaces the logo with an uploaded image.
func (s *Server) putLogo(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxLogo))
	if err != nil {
		fail(w, fmt.Errorf("the logo must be at most 1 MB: %w", err))
		return
	}
	ext := logoKind(b)
	if ext == "" {
		fail(w, errors.New("not an image: use PNG, SVG, JPEG, WebP or GIF"))
		return
	}
	if err := writeFileAtomic(filepath.Join(s.DataDir, "logo."+ext), b); err != nil {
		fail(w, err)
		return
	}
	for e := range logoTypes {
		if e != ext {
			os.Remove(filepath.Join(s.DataDir, "logo."+e))
		}
	}
	log.Printf("logo: replaced (%s, %d bytes)", ext, len(b))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "type": ext})
}

// deleteLogo goes back to the built-in logo.
func (s *Server) deleteLogo(w http.ResponseWriter, r *http.Request) {
	for e := range logoTypes {
		os.Remove(filepath.Join(s.DataDir, "logo."+e))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
