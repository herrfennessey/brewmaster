package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/herrfennessey/brewmaster/api/internal/ai"
)

const roastDateLayout = "2006-01-02"

// ParseRoastDateHandler handles POST /api/parse-roast-date requests. It exists
// so the user can answer the post-parse "when were these roasted?" prompt in
// natural language (including expiration dates and relatives like "2 weeks
// ago") and have the LLM convert it to an ISO date for the rule engine.
type ParseRoastDateHandler struct {
	provider ai.Provider
}

// NewParseRoastDateHandler creates a ParseRoastDateHandler.
func NewParseRoastDateHandler(provider ai.Provider) http.Handler {
	return &ParseRoastDateHandler{provider: provider}
}

type parseRoastDateRequest struct {
	Text string `json:"text"`
}

type parseRoastDateResponse struct {
	RoastDate *string `json:"roast_date"`
	Reasoning string  `json:"reasoning"`
}

func (h *ParseRoastDateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	var req parseRoastDateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		writeError(w, http.StatusBadRequest, "text must not be empty")
		return
	}
	annotateParseSpan(r.Context(),
		attribute.Int("parse_roast_date.input.length", len(text)),
	)

	today := time.Now().UTC().Format(roastDateLayout)
	raw, err := h.provider.Complete(r.Context(), &ai.CompletionRequest{
		SystemPrompt:  ai.ParseRoastDatePrompt,
		UserMessage:   fmt.Sprintf("Today's date: %s\n\nUser input: %s", today, text),
		MaxTokens:     200,
		Tool:          ai.ParseRoastDateTool,
		Deterministic: true,
		Phase:         "parse_roast_date",
	})
	if err != nil {
		slog.Error("AI roast-date completion failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to parse roast date")
		return
	}

	var resp parseRoastDateResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		slog.Error("failed to parse AI roast-date response", "error", err, "raw", truncate(raw, 200))
		writeError(w, http.StatusInternalServerError, "AI returned an unexpected response format")
		return
	}

	// The schema asks for ISO format, but the model can still produce malformed
	// strings under certain prompts. Reject here using the same parser the rule
	// engine relies on (brew.Normalize), so what we return is guaranteed valid
	// upstream.
	if resp.RoastDate != nil {
		if _, err := time.Parse(roastDateLayout, *resp.RoastDate); err != nil {
			slog.Warn("AI returned non-ISO roast date, dropping", "raw", *resp.RoastDate, "error", err)
			resp.RoastDate = nil
			// Stale reasoning at this point would claim a date was extracted; the
			// client renders it as the failure message, so replace it.
			resp.Reasoning = "I couldn't read a clear roast date from that — try a different phrasing or skip."
		}
	}

	annotateParseSpan(r.Context(),
		attribute.Bool("parse_roast_date.resolved", resp.RoastDate != nil),
	)
	// Reasoning is the model's free-text rationale and can echo user input, so
	// keep it out of structured logs. The "resolved" boolean is enough for
	// telemetry; the full reasoning still goes back to the client.
	slog.InfoContext(r.Context(), "roast date parsed",
		"resolved", resp.RoastDate != nil,
	)
	writeJSON(w, http.StatusOK, resp)
}
