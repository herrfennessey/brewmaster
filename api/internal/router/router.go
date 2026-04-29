// Package router sets up HTTP routing for the brewmaster API.
package router

import (
	"net/http"
	"os"
	"strings"

	"github.com/herrfennessey/brewmaster/api/internal/ai"
	"github.com/herrfennessey/brewmaster/api/internal/handler"
)

// New creates and returns the HTTP router with all routes configured.
func New(provider ai.Provider) http.Handler {
	mux := http.NewServeMux()

	// Public health check
	mux.HandleFunc("GET /health", handler.Health)

	// AI endpoints
	mux.Handle("POST /api/parse-bean", handler.NewParseHandler(provider))
	mux.Handle("POST /api/generate-parameters", handler.NewBrewHandler(provider))

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
		f, err := root.Open(r.URL.Path)
		if os.IsNotExist(err) {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		if err != nil || f == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		defer f.Close() //nolint:errcheck

		stat, statErr := f.Stat()
		if statErr != nil || !stat.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Path is a directory — fall back to index.html if no directory index exists.
		indexPath := strings.TrimSuffix(r.URL.Path, "/") + "/index.html"
		indexFile, openErr := root.Open(indexPath)
		if openErr == nil {
			defer indexFile.Close() //nolint:errcheck
		}
		if os.IsNotExist(openErr) {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

// corsMiddleware adds permissive CORS headers.
// This is a personal tool with no auth in Phase 1; the open wildcard is accepted risk.
// TODO: restrict Allow-Origin to known domains before sharing this with others.
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
