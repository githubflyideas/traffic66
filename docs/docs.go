// Package docs carries the user manual (docs/MANUAL*.md and its images)
// into the program, for the Help page of the web UI. The same files are
// the manual on GitHub.
package docs

import "embed"

//go:embed MANUAL*.md images/*.png
var FS embed.FS
