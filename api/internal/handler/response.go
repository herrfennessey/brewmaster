package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

// stripCodeFences removes markdown JSON code fences that some models add despite being told not to.
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		end := strings.LastIndex(s, "```")
		if end > 3 {
			s = s[3:end]
		}
		// remove optional language identifier on the opening fence line
		if nl := strings.Index(s, "\n"); nl != -1 {
			s = s[nl+1:]
		}
	}
	return strings.TrimSpace(s)
}
