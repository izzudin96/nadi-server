// Package webui embeds the built Vue dashboard so the server ships as a single
// binary. Build the frontend into ./dist with `make web` before a release
// build. The dist/ directory carries a .gitkeep so this package always
// compiles, even before the frontend has been built.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// FS returns the dashboard files rooted at dist/.
func FS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
