package api

import (
	"io/fs"
	"net/http"
	"path"
	"regexp"

	"github.com/githubflyideas/traffic66/docs"
)

var manualLang = regexp.MustCompile(`^[a-z]{2}$`)

// manual serves the user manual in the language asked for, else English.
// It is markdown; the page renders it, escaping everything in it.
func (s *Server) manual(w http.ResponseWriter, r *http.Request) {
	name := "MANUAL.md"
	if l := r.URL.Query().Get("lang"); manualLang.MatchString(l) && l != "en" {
		if _, err := fs.Stat(docs.FS, "MANUAL."+l+".md"); err == nil {
			name = "MANUAL." + l + ".md"
		}
	}
	b, err := fs.ReadFile(docs.FS, name)
	if err != nil {
		http.Error(w, "no manual", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(b)
}

// manualImage serves one of the manual's screenshots.
func (s *Server) manualImage(w http.ResponseWriter, r *http.Request) {
	name := path.Base(r.PathValue("name"))
	b, err := fs.ReadFile(docs.FS, "images/"+name)
	if err != nil || path.Ext(name) != ".png" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(b)
}
