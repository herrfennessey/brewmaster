// Package handler contains HTTP request handlers.
package handler

import (
	"encoding/json"
	"net/http"
)

// Health returns a simple liveness response.
func Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
