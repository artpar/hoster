// Package api provides embedded Web UI serving.
// Following the same pattern as APIGate for consistency.
package api

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

//go:embed all:webui/dist
var webUIAssets embed.FS

// WebUIHandler returns an HTTP handler that serves the embedded Web UI.
// It serves static files and falls back to index.html for SPA routing.
// Prefers web/dist/ on disk (always fresh from Vite) over embedded assets.
func WebUIHandler() http.Handler {
	// Prefer filesystem web/dist/ — always up-to-date during development.
	// Falls back to embedded assets when the directory doesn't exist (production binary).
	var distFS fs.FS
	if info, err := os.Stat("web/dist"); err == nil && info.IsDir() {
		distFS = os.DirFS("web/dist")
	} else if sub, err := fs.Sub(webUIAssets, "webui/dist"); err == nil {
		distFS = sub
	} else {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Hoster UI Not Built</title></head>
<body style="font-family: system-ui; padding: 2rem; max-width: 600px; margin: 0 auto;">
<h1>Hoster UI Not Built</h1>
<p>The web UI assets have not been built yet. To build them:</p>
<pre style="background: #f4f4f4; padding: 1rem; border-radius: 4px;">
cd web
npm install
npm run build
</pre>
<p>Then rebuild the Hoster binary to embed the assets.</p>
</body>
</html>`))
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get path and clean it
		urlPath := r.URL.Path
		urlPath = path.Clean(urlPath)
		if urlPath == "/" || urlPath == "" || urlPath == "." {
			urlPath = "index.html"
		} else {
			urlPath = strings.TrimPrefix(urlPath, "/")
		}

		// Try to serve the requested file
		content, err := fs.ReadFile(distFS, urlPath)
		if err == nil {
			// Set content type based on extension
			contentType := "application/octet-stream"
			switch {
			case strings.HasSuffix(urlPath, ".html"):
				contentType = "text/html; charset=utf-8"
			case strings.HasSuffix(urlPath, ".js"):
				contentType = "application/javascript"
			case strings.HasSuffix(urlPath, ".css"):
				contentType = "text/css"
			case strings.HasSuffix(urlPath, ".svg"):
				contentType = "image/svg+xml"
			case strings.HasSuffix(urlPath, ".json"):
				contentType = "application/json"
			case strings.HasSuffix(urlPath, ".png"):
				contentType = "image/png"
			case strings.HasSuffix(urlPath, ".ico"):
				contentType = "image/x-icon"
			case strings.HasSuffix(urlPath, ".woff"):
				contentType = "font/woff"
			case strings.HasSuffix(urlPath, ".woff2"):
				contentType = "font/woff2"
			}
			w.Header().Set("Content-Type", contentType)
			w.Write(content)
			return
		}

		// Check if it's an asset request (has file extension) - return 404
		if strings.Contains(path.Base(urlPath), ".") {
			http.NotFound(w, r)
			return
		}

		// SPA fallback: serve index.html for client-side routing
		content, err = fs.ReadFile(distFS, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(content)
	})
}

// IsWebUIBuilt checks if the web UI has been built.
func IsWebUIBuilt() bool {
	if info, err := os.Stat("web/dist/index.html"); err == nil && !info.IsDir() {
		return true
	}
	_, err := fs.Stat(webUIAssets, "webui/dist/index.html")
	return err == nil
}
