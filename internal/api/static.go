package api

import (
	"io/fs"
	"net/http"
	"strings"
)

// spaHandler serves the embedded dashboard with an SPA fallback: any path that
// is not an existing file (e.g. a client-side route like /devices/x) is served
// index.html instead of a 404.
func spaHandler(fsys fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			fileServer.ServeHTTP(w, r)
			return
		}
		if _, err := fs.Stat(fsys, path); err != nil {
			r.URL.Path = "/" // SPA route: fall back to index.html
		}
		fileServer.ServeHTTP(w, r)
	}
}
