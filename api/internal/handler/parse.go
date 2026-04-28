package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/herrfennessey/brewmaster/api/internal/ai"
	"github.com/herrfennessey/brewmaster/api/internal/models"
)

// ParseHandler handles POST /api/parse-bean requests.
type ParseHandler struct {
	provider ai.Provider
}

// NewParseHandler creates a ParseHandler with the given AI provider.
func NewParseHandler(provider ai.Provider) http.Handler {
	return &ParseHandler{provider: provider}
}

type parseBeanRequest struct {
	InputType string `json:"input_type"`
	Content   string `json:"content"`
}

type parsedAIResponse struct {
	Confidence models.BeanConfidence `json:"confidence"`
	Parsed     models.ParsedBean     `json:"parsed"`
}

func (h *ParseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req parseBeanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.InputType != "text" {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported input_type %q: only \"text\" is supported", req.InputType))
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content must not be empty")
		return
	}

	raw, err := h.provider.Complete(r.Context(), &ai.CompletionRequest{
		SystemPrompt: ai.ParseBeanPrompt,
		UserMessage:  req.Content,
		MaxTokens:    1024,
		Tool:         ai.ParseBeanTool,
	})
	if err != nil {
		slog.Error("AI completion failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to parse bean info")
		return
	}

	var aiResp parsedAIResponse
	if err := json.Unmarshal([]byte(raw), &aiResp); err != nil {
		slog.Error("failed to parse AI tool response", "error", err, "raw", raw)
		writeError(w, http.StatusInternalServerError, "AI returned an unexpected response format")
		return
	}

	profile := models.BeanProfile{
		ID:         uuid.NewString(),
		SourceType: "text",
		Parsed:     aiResp.Parsed,
		Confidence: aiResp.Confidence,
		CreatedAt:  time.Now().UTC(),
	}

	writeJSON(w, http.StatusOK, profile)
}
