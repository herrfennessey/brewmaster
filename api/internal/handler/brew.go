package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/herrfennessey/brewmaster/api/internal/ai"
	"github.com/herrfennessey/brewmaster/api/internal/brew"
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
	ExtractionMethod string             `json:"extraction_method"`
	DrinkType        string             `json:"drink_type"`
	BeanProfile      models.BeanProfile `json:"bean_profile"`
}

// drinksByMethod is the canonical valid (extraction_method → drink_type) set.
// Both client and server must agree on this — keep in sync with pwa/src/screens/Home.tsx.
var drinksByMethod = map[string]map[string]bool{
	"espresso": {
		"espresso": true, "americano": true, "macchiato": true,
		"cortado": true, "cappuccino": true, "flat white": true, "latte": true,
	},
	"pourover": {
		"black": true, "cafe au lait": true,
	},
}

type brewAnnotation struct {
	Reasoning string   `json:"reasoning"`
	Flags     []string `json:"flags"`
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
	if msg := normalizeBrewRequest(&req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	canonical := brew.Normalize(&req.BeanProfile.Parsed)
	computed := brew.ComputeParams(&canonical, req.ExtractionMethod, req.DrinkType)
	suit := brew.ComputeSuitability(&canonical, req.DrinkType)
	conf := brew.ComputeConfidence(&canonical)

	annotateBrewSpan(r.Context(), &req, &canonical, &computed, suit, conf)

	slog.InfoContext(r.Context(), "brew computed",
		"version", computed.Version,
		"applied_rules", computed.AppliedRules,
		"suitability_rule", suit.Rule,
		"confidence", conf.Level,
	)

	reasoning, flags := h.annotate(r, &req, &computed, suit, conf)

	params := models.BrewParameters{
		BeanID:           req.BeanProfile.ID,
		ExtractionMethod: req.ExtractionMethod,
		DrinkType:        req.DrinkType,
		Parameters:       computed.Params,
		Confidence:       conf,
		Suitability:      suit.DrinkSuitability,
		Reasoning:        reasoning,
		Flags:            flags,
		Iteration:        1,
	}
	writeJSON(w, http.StatusOK, params)
}

// annotate calls the LLM for optional prose annotation; falls back to a
// deterministic summary if the call fails or returns empty reasoning.
func (h *BrewHandler) annotate(r *http.Request, req *generateParamsRequest, computed *brew.ComputedParams, suit brew.SuitabilityResult, conf models.BrewConfidence) (reasoning string, flags []string) {
	fallback := brew.FallbackReasoning(computed.AppliedRules)

	raw, err := h.provider.Complete(r.Context(), &ai.CompletionRequest{
		SystemPrompt:  ai.BrewAnnotatePrompt,
		UserMessage:   buildBrewAnnotateMessage(req, computed, suit, conf),
		MaxTokens:     400,
		Tool:          ai.BrewAnnotateTool,
		Deterministic: true,
	})
	if err != nil {
		slog.Error("brew annotation failed, using fallback", "error", err)
		return fallback, nil
	}

	var ann brewAnnotation
	if err := json.Unmarshal([]byte(raw), &ann); err != nil {
		slog.Error("failed to parse brew annotation", "error", err, "raw", truncate(raw, 200))
		return fallback, nil
	}
	if ann.Reasoning == "" {
		ann.Reasoning = fallback
	}
	return ann.Reasoning, ann.Flags
}

// normalizeBrewRequest lowercases and validates extraction_method/drink_type.
// Returns a non-empty message describing why the request is invalid, or ""
// when the request is valid (and possibly mutated to fill defaults).
func normalizeBrewRequest(req *generateParamsRequest) string {
	req.ExtractionMethod = strings.ToLower(strings.TrimSpace(req.ExtractionMethod))
	req.DrinkType = strings.ToLower(strings.TrimSpace(req.DrinkType))

	if req.ExtractionMethod == "" {
		req.ExtractionMethod = "espresso"
	}
	drinks, ok := drinksByMethod[req.ExtractionMethod]
	if !ok {
		return fmt.Sprintf("extraction_method %q is not supported (use 'espresso' or 'pourover')", req.ExtractionMethod)
	}

	if req.DrinkType == "" {
		if req.ExtractionMethod == "pourover" {
			req.DrinkType = "black"
		} else {
			req.DrinkType = "espresso"
		}
		return ""
	}
	if !drinks[req.DrinkType] {
		return fmt.Sprintf("drink_type %q is not valid for extraction_method %q", req.DrinkType, req.ExtractionMethod)
	}
	return ""
}

// annotateBrewSpan attaches brew-specific metadata to the active request span
// (typically the otelhttp-created span). Safe to call when telemetry is disabled.
func annotateBrewSpan(ctx context.Context, req *generateParamsRequest, bean *brew.CanonicalBean, computed *brew.ComputedParams, suit brew.SuitabilityResult, conf models.BrewConfidence) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}
	rules := make([]string, 0, len(computed.AppliedRules))
	for _, r := range computed.AppliedRules {
		rules = append(rules, string(r))
	}
	span.SetAttributes(
		attribute.String("brew.method", req.ExtractionMethod),
		attribute.String("brew.drink", req.DrinkType),
		attribute.String("brew.ruleset_version", computed.Version),
		attribute.String("brew.suitability.level", suit.Level),
		attribute.String("brew.suitability.rule", string(suit.Rule)),
		attribute.String("brew.confidence.level", conf.Level),
		attribute.Int("brew.applied_rules.count", len(rules)),
		attribute.String("brew.applied_rules", strings.Join(rules, ",")),
		attribute.String("brew.bean.process", bean.Process),
		attribute.String("brew.bean.roast_level", bean.RoastLevel),
		attribute.String("brew.bean.origin_country", bean.OriginCountry),
		attribute.String("brew.bean.varietal", bean.Varietal),
		attribute.Bool("brew.bean.altitude_known", bean.AltitudeKnown),
		attribute.Float64("brew.params.temp_c", computed.Params.TempC.Value),
		attribute.String("brew.params.ratio", computed.Params.Ratio),
	)
	if bean.AltitudeKnown {
		span.SetAttributes(attribute.Float64("brew.bean.altitude_m", bean.AltitudeM))
	}
}

func buildBrewAnnotateMessage(req *generateParamsRequest, computed *brew.ComputedParams, suit brew.SuitabilityResult, conf models.BrewConfidence) string {
	beanJSON, err := json.Marshal(req.BeanProfile.Parsed)
	if err != nil {
		panic("brewmaster: marshal BeanProfile.Parsed: " + err.Error())
	}
	p := computed.Params
	return fmt.Sprintf(
		"Extraction: %s\nDrink: %s\n\nBean profile:\n%s\n\nComputed parameters:\nDose %.1fg | Yield %.1fg | Ratio %s\nTemp %.1f°C | Time %.0fs | Bloom/Preinfusion %.0fs\n\nSuitability: %s — %s\nConfidence: %s\nRules applied: %v",
		req.ExtractionMethod, req.DrinkType,
		string(beanJSON),
		p.DoseG.Value, p.YieldG.Value, p.Ratio,
		p.TempC.Value, p.TimeS.Value, p.PreinfusionS.Value,
		suit.Level, suit.Reason,
		conf.Level,
		computed.AppliedRules,
	)
}
