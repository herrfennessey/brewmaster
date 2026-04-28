// Package router sets up HTTP routing for the brewmaster API.
package router

import (
	"net/http"
	"os"
	"strings"

	"github.com/herrfennessey/brewmaster/api/internal/handler"
)

// New creates and returns the HTTP router with all routes configured.
func New() http.Handler {
	mux := http.NewServeMux()

	// Public health check
	mux.HandleFunc("GET /health", handler.Health)

	// Serve the React PWA from ./static if it has been built.
	// Falls back to the health handler when running without a built frontend
	// (e.g., during `go test` or early dev iterations).
	staticDir := "static"
	if _, err := os.Stat(staticDir); err == nil {
		mux.Handle("/", spaHandler(http.Dir(staticDir)))
	} else {
		mux.HandleFunc("/", handler.Health)
	}

	return corsMiddleware(mux)
}

// spaHandler serves static files and falls back to index.html for client-side routing.
func spaHandler(root http.FileSystem) http.Handler {
	fileServer := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		f, err := root.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				r.URL.Path = "/"
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		if f != nil {
			defer f.Close() //nolint:errcheck
			stat, statErr := f.Stat()
			if statErr == nil && stat.IsDir() {
				indexPath := strings.TrimSuffix(path, "/") + "/index.html"
				if _, openErr := root.Open(indexPath); os.IsNotExist(openErr) {
					r.URL.Path = "/"
					fileServer.ServeHTTP(w, r)
					return
				}
			}
		}

		fileServer.ServeHTTP(w, r)
	})
}

// corsMiddleware adds permissive CORS headers for local development.
// In production this is safe because Cloud Run sits behind Google's infrastructure
// and auth will be enforced at the application layer.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
