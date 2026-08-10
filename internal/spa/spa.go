// Package spa serves the built React console.
//
// The bundle can come from three places: embedded in the binary (build with
// -tags embed_spa), read from disk via SPA_DIR, or not at all, in which case a
// placeholder explains how to build it. The build tag matters because
// //go:embed fails at compile time when the directory is missing, and that would
// break `go build ./...` for anyone who has not run npm first.
package spa

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"go-ledger/internal/httpx"
)

// Resolve picks a source for the built assets. dir wins when set, so a
// production bundle can be smoke-tested without rebuilding the binary.
func Resolve(dir string) (fs.FS, bool) {
	if dir != "" {
		if _, err := os.Stat(path.Join(dir, "index.html")); err == nil {
			return os.DirFS(dir), true
		}
		return nil, false
	}

	return embedded()
}

// Handler serves the SPA with a history fallback: any path that is not a real
// file returns index.html with a 200, because the client-side router owns those
// routes. API paths never reach here, since /api and /ops set their own JSON
// NotFound handlers before being mounted.
func Handler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" || name == "." {
			name = "index.html"
		}

		file, err := fsys.Open(name)
		if err != nil {
			serveIndex(w, fsys)
			return
		}

		info, statErr := file.Stat()
		file.Close()

		if statErr != nil || info.IsDir() {
			serveIndex(w, fsys)
			return
		}

		// Vite emits content-hashed filenames under assets/, so those are safe
		// to cache forever. index.html must never be cached or a deploy would
		// keep pointing at the old bundle.
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		fileServer.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, fsys fs.FS) {
	file, err := fsys.Open("index.html")
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, file)
}

// PlaceholderHandler stands in when no bundle is available, so hitting the root
// explains what to do instead of returning a bare 404.
func PlaceholderHandler() http.Handler {
	const page = `<!doctype html>
<meta charset="utf-8">
<title>go-ledger</title>
<style>body{font-family:system-ui,sans-serif;max-width:40rem;margin:4rem auto;padding:0 1rem;line-height:1.6}code{background:#eee;padding:.15rem .35rem;border-radius:4px}</style>
<h1>go-ledger API is running</h1>
<p>The React console is not bundled into this binary. Either:</p>
<ul>
  <li>run the dev server: <code>cd web &amp;&amp; npm run dev</code> (then use <a href="http://localhost:5173">localhost:5173</a>), or</li>
  <li>build it and point at the output: <code>cd web &amp;&amp; npm run build</code>, then start the server with <code>SPA_DIR=internal/spa/dist</code>, or</li>
  <li>build a single binary: <code>go build -tags embed_spa ./cmd/server</code>.</li>
</ul>
<p>The API itself is live at <code>/api</code> and <code>/ops</code>.</p>`

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, page)
	})
}
