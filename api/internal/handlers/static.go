package handlers

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// staticHandler serves a built single-page app from dir for every request the API routes
// did not claim. Unknown extension-less paths (client-side routes such as /t/thr_abc) get
// index.html so deep links and refreshes work; unknown files with an extension, anything
// under /api, and non-GET/HEAD methods get notFound instead of the HTML shell.
func staticHandler(dir string, notFound gin.HandlerFunc) gin.HandlerFunc {
	// os.DirFS refuses names that are not valid, slash-separated and rooted inside dir.
	fsys := os.DirFS(dir)

	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			notFound(c)
			return
		}
		// Clean against a rooted path so ".." can never climb above the static directory.
		clean := path.Clean("/" + c.Request.URL.Path)
		if clean == "/api" || strings.HasPrefix(clean, "/api/") {
			notFound(c)
			return
		}
		name := strings.TrimPrefix(clean, "/")
		if name == "" {
			name = "index.html"
		}

		if serveStaticFile(c, fsys, name) {
			return
		}
		if path.Ext(name) != "" { // a missing asset must not be answered with HTML
			notFound(c)
			return
		}
		if serveStaticFile(c, fsys, "index.html") {
			return
		}
		notFound(c)
	}
}

// serveStaticFile writes name from fsys if it is a regular file, reporting whether it did.
// Directories are deliberately not served, so there is never a file listing.
func serveStaticFile(c *gin.Context, fsys fs.FS, name string) bool {
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }() // read-only file: nothing useful to do with a close error
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return false
	}

	rs, ok := f.(io.ReadSeeker)
	if !ok {
		data, err := io.ReadAll(f)
		if err != nil {
			return false
		}
		rs = bytes.NewReader(data)
	}

	// Vite fingerprints everything under assets/, so those can be cached forever. Anything
	// else (notably index.html) must revalidate so a new deploy is picked up immediately.
	if strings.HasPrefix(name, "assets/") {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "no-cache")
	}
	http.ServeContent(c.Writer, c.Request, name, info.ModTime(), rs)
	return true
}
