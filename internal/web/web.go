// Package web embeds the browser UI and the translation files.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var files embed.FS

// FS is the web root.
func FS() fs.FS {
	sub, _ := fs.Sub(files, "static")
	return sub
}

// CSP is the Content-Security-Policy of the web UI's pages. Scripts come
// only from the program's own files (index.html has no inline script and
// no inline event handlers, which the test checks); the pages build markup
// with style attributes, so inline styles stay allowed; images are the
// program's own (the logo), data: (the watermark and icons) and blob:.
// Nothing is loaded from another host. frame-ancestors is left out on
// purpose: the embed pages are meant to be framed by other platforms.
const CSP = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'"
