package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"

	"github.com/herrfennessey/brewmaster/api/internal/ai"
	"github.com/herrfennessey/brewmaster/api/internal/models"
)

// ParseHandler handles POST /api/parse-bean requests.
type ParseHandler struct {
	provider   ai.Provider
	httpClient *http.Client
}

// NewParseHandler creates a ParseHandler with the given AI provider.
func NewParseHandler(provider ai.Provider) http.Handler {
	return &ParseHandler{
		provider:   provider,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type parseBeanRequest struct {
	InputType string `json:"input_type"`
	Content   string `json:"content"`
}

type parsedAIResponse struct {
	Confidence models.BeanConfidence `json:"confidence"`
	Parsed     models.ParsedBean     `json:"parsed"`
}

var wsRe = regexp.MustCompile(`\s+`)

func (h *ParseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		h.handleImage(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req parseBeanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	switch req.InputType {
	case "text":
		h.handleText(w, r, req)
	case "url":
		h.handleURL(w, r, req)
	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported input_type %q", req.InputType))
	}
}

func (h *ParseHandler) handleText(w http.ResponseWriter, r *http.Request, req parseBeanRequest) {
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
	h.writeProfile(w, raw, "text")
}

func (h *ParseHandler) handleImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read uploaded file")
		return
	}
	mediaType := http.DetectContentType(data)
	switch mediaType {
	case "image/jpeg", "image/png", "image/webp":
	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported image type %q: use JPEG, PNG, or WEBP", mediaType))
		return
	}
	slog.Info("image uploaded", "media_type", mediaType, "size_bytes", len(data))

	slog.Info("calling vision API")
	raw, err := h.provider.CompleteWithImage(r.Context(), &ai.CompletionRequest{
		SystemPrompt: ai.ParseBeanPrompt,
		UserMessage:  "Extract the bean profile from this image of a coffee bag label.",
		MaxTokens:    1024,
		Tool:         ai.ParseBeanTool,
	}, data, mediaType)
	if err != nil {
		slog.Error("AI vision completion failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to parse bean info from image")
		return
	}

	var imageResp parsedAIResponse
	if err := json.Unmarshal([]byte(raw), &imageResp); err != nil {
		slog.Error("failed to parse AI image response", "error", err, "raw", truncate(raw, 200))
		writeError(w, http.StatusInternalServerError, "AI returned an unexpected response format")
		return
	}
	slog.Info("image parsed",
		"roaster", derefStr(imageResp.Parsed.RoasterName),
		"producer", derefStr(imageResp.Parsed.Producer),
		"origin", derefStr(imageResp.Parsed.OriginCountry),
		"confidence", imageResp.Confidence.Level,
	)

	merged, sourceType := h.enrichImageProfile(r, &imageResp)
	slog.Info("image parse complete", "source_type", sourceType)
	h.writeProfileDirect(w, &merged, sourceType)
}

func (h *ParseHandler) enrichImageProfile(r *http.Request, imageResp *parsedAIResponse) (result parsedAIResponse, sourceType string) {
	roasterName := imageResp.Parsed.RoasterName
	if roasterName == nil {
		slog.Info("skipping web enrichment: no roaster name in image result")
		return *imageResp, "image"
	}
	hint := buildHint(&imageResp.Parsed)
	slog.Info("starting web enrichment", "roaster", *roasterName, "hint", hint)

	webText, err := h.provider.FindRoasterContent(r.Context(), *roasterName, hint)
	if err != nil {
		slog.Warn("web enrichment failed, using image-only result", "error", err)
		return *imageResp, "image"
	}
	slog.Info("web search returned content", "chars", len(webText), "preview", truncate(webText, 120))

	webRaw, err := h.provider.Complete(r.Context(), &ai.CompletionRequest{
		SystemPrompt: ai.ParseBeanPrompt,
		UserMessage:  webText,
		MaxTokens:    1024,
		Tool:         ai.ParseBeanTool,
	})
	if err != nil {
		slog.Warn("web re-parse failed, using image-only result", "error", err)
		return *imageResp, "image"
	}
	var webResp parsedAIResponse
	if err := json.Unmarshal([]byte(webRaw), &webResp); err != nil {
		slog.Warn("web response unmarshal failed, using image-only result", "error", err)
		return *imageResp, "image"
	}
	slog.Info("web parse complete",
		"varietal", derefStr(webResp.Parsed.Varietal),
		"process", derefStr(webResp.Parsed.Process),
		"altitude_m", webResp.Parsed.AltitudeM,
		"flavor_notes", webResp.Parsed.FlavorNotes,
		"confidence", webResp.Confidence.Level,
	)
	return mergeProfiles(imageResp, &webResp), "image+web"
}

func derefStr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func buildHint(p *models.ParsedBean) string {
	var parts []string
	if p.OriginCountry != nil {
		parts = append(parts, *p.OriginCountry)
	}
	if p.OriginRegion != nil {
		parts = append(parts, *p.OriginRegion)
	}
	if p.Producer != nil {
		parts = append(parts, *p.Producer)
	}
	return strings.Join(parts, ", ")
}

func mergeProfiles(img, web *parsedAIResponse) parsedAIResponse {
	merged := *img
	p := &merged.Parsed
	w := &web.Parsed

	if p.Producer == nil {
		p.Producer = w.Producer
	}
	if p.OriginCountry == nil {
		p.OriginCountry = w.OriginCountry
	}
	if p.OriginRegion == nil {
		p.OriginRegion = w.OriginRegion
	}
	if p.AltitudeM == nil {
		p.AltitudeM = w.AltitudeM
	}
	if p.AltitudeConfidence == nil {
		p.AltitudeConfidence = w.AltitudeConfidence
	}
	if p.Varietal == nil {
		p.Varietal = w.Varietal
	}
	if p.Process == nil {
		p.Process = w.Process
	}
	if p.RoastLevel == nil {
		p.RoastLevel = w.RoastLevel
	}
	if p.RoastDate == nil {
		p.RoastDate = w.RoastDate
	}
	if p.RoasterName == nil {
		p.RoasterName = w.RoasterName
	}
	if p.LotYear == nil {
		p.LotYear = w.LotYear
	}

	p.FlavorNotes = mergeFlavors(p.FlavorNotes, w.FlavorNotes)

	if merged.Confidence.Level == "low" {
		merged.Confidence.Level = "medium"
	}
	merged.Confidence.Notes += " (enriched from roaster website)"
	return merged
}

func mergeFlavors(base, extra []string) []string {
	seen := make(map[string]bool, len(base))
	for _, n := range base {
		seen[n] = true
	}
	for _, n := range extra {
		if !seen[n] {
			base = append(base, n)
		}
	}
	return base
}

var blockedHosts = []string{"localhost", "127.", "::1", "0.0.0.0", "169.254."}

func (h *ParseHandler) handleURL(w http.ResponseWriter, r *http.Request, req parseBeanRequest) {
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content must not be empty")
		return
	}
	parsed, err := url.Parse(req.Content)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		writeError(w, http.StatusBadRequest, "content must be a valid http or https URL")
		return
	}
	for _, blocked := range blockedHosts {
		if strings.Contains(parsed.Hostname(), blocked) {
			writeError(w, http.StatusBadRequest, "URL host is not allowed")
			return
		}
	}

	resp, err := h.httpClient.Get(req.Content) //nolint:noctx
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch URL")
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to parse page content")
		return
	}
	doc.Find("script, style, nav, footer, header").Remove()
	text := wsRe.ReplaceAllString(doc.Text(), " ")
	if len(text) > 32000 {
		text = text[:32000]
	}

	raw, err := h.provider.Complete(r.Context(), &ai.CompletionRequest{
		SystemPrompt: ai.ParseBeanPrompt,
		UserMessage:  text,
		MaxTokens:    1024,
		Tool:         ai.ParseBeanTool,
	})
	if err != nil {
		slog.Error("AI completion failed for URL", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to parse bean info from URL")
		return
	}
	h.writeProfile(w, raw, "url")
}

func (h *ParseHandler) writeProfile(w http.ResponseWriter, raw, sourceType string) {
	var aiResp parsedAIResponse
	if err := json.Unmarshal([]byte(raw), &aiResp); err != nil {
		slog.Error("failed to parse AI tool response", "error", err, "raw", truncate(raw, 200))
		writeError(w, http.StatusInternalServerError, "AI returned an unexpected response format")
		return
	}
	h.writeProfileDirect(w, &aiResp, sourceType)
}

func (h *ParseHandler) writeProfileDirect(w http.ResponseWriter, aiResp *parsedAIResponse, sourceType string) {
	profile := models.BeanProfile{
		ID:         uuid.NewString(),
		SourceType: sourceType,
		Parsed:     aiResp.Parsed,
		Confidence: aiResp.Confidence,
		CreatedAt:  time.Now().UTC(),
	}
	writeJSON(w, http.StatusOK, profile)
}
