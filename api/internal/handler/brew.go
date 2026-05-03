package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"

	"github.com/herrfennessey/brewmaster/api/internal/ai"
	"github.com/herrfennessey/brewmaster/api/internal/models"
)

// BrewHandler handles POST /api/generate-parameters requests.
type BrewHandler struct {
	provider ai.Provider
}

// NewBrewHandler creates a BrewHandler with the given AI provider.
func NewBrewHandler(provider ai.Provider) http.Handler {
	return &BrewHandler{provider: provider}
}

type generateParamsRequest struct {
	TargetDrink      string             `json:"target_drink"`
	ExtractionMethod string             `json:"extraction_method"`
	DrinkType        string             `json:"drink_type"`
	BeanProfile      models.BeanProfile `json:"bean_profile"`
}

type brewAIResponse struct {
	Suitability models.DrinkSuitability `json:"suitability"`
	Confidence  models.BrewConfidence   `json:"confidence"`
	Reasoning   string                  `json:"reasoning"`
	Flags       []string                `json:"flags"`
	Parameters  models.BrewParams       `json:"parameters"`
	Iteration   int                     `json:"iteration"`
}

func (h *BrewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req generateParamsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.BeanProfile.ID == "" {
		writeError(w, http.StatusBadRequest, "bean_profile must be provided with a valid id")
		return
	}
	normalizeBrewRequest(&req)

	userMsg := buildBrewUserMessage(&req)

	raw, err := h.provider.Complete(r.Context(), &ai.CompletionRequest{
		SystemPrompt: ai.BrewParamPrompt,
		UserMessage:  userMsg,
		MaxTokens:    1024,
		Tool:         ai.BrewParamTool,
	})
	if err != nil {
		slog.Error("AI completion failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to generate brew parameters")
		return
	}

	var aiResp brewAIResponse
	if err := json.Unmarshal([]byte(raw), &aiResp); err != nil {
		slog.Error("failed to parse AI tool response", "error", err, "raw", truncate(raw, 200))
		writeError(w, http.StatusInternalServerError, "AI returned an unexpected response format")
		return
	}

	fixRatio(&aiResp.Parameters)

	params := models.BrewParameters{
		BeanID:           req.BeanProfile.ID,
		ExtractionMethod: req.ExtractionMethod,
		DrinkType:        req.DrinkType,
		Parameters:       aiResp.Parameters,
		Confidence:       aiResp.Confidence,
		Suitability:      aiResp.Suitability,
		Reasoning:        aiResp.Reasoning,
		Flags:            aiResp.Flags,
		Iteration:        aiResp.Iteration,
	}
	if params.Iteration == 0 {
		params.Iteration = 1
	}

	writeJSON(w, http.StatusOK, params)
}

func normalizeBrewRequest(req *generateParamsRequest) {
	if req.ExtractionMethod == "" {
		if req.TargetDrink == "pourover" || req.TargetDrink == "filter" {
			req.ExtractionMethod = "pourover"
		} else {
			req.ExtractionMethod = "espresso"
		}
	}
	if req.DrinkType == "" {
		if req.ExtractionMethod == "pourover" {
			req.DrinkType = "black"
		} else {
			req.DrinkType = "espresso"
		}
	}
}

// fixRatio overwrites the AI-provided ratio string with the value derived from
// yield_g / dose_g so the three fields are always internally consistent.
func fixRatio(p *models.BrewParams) {
	if p.DoseG.Value == 0 {
		return
	}
	computed := p.YieldG.Value / p.DoseG.Value
	rounded := math.Round(computed*10) / 10
	p.Ratio = fmt.Sprintf("1:%.1f", rounded)
}

func buildBrewUserMessage(req *generateParamsRequest) string {
	// BeanProfile contains no unencodable types; marshal cannot fail.
	beanJSON, err := json.Marshal(req.BeanProfile)
	if err != nil {
		panic("brewmaster: marshal BeanProfile: " + err.Error())
	}
	return fmt.Sprintf("Extraction method: %s\nDrink: %s\n\nBean profile:\n%s",
		req.ExtractionMethod, req.DrinkType, string(beanJSON))
}
