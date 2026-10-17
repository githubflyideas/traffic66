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
