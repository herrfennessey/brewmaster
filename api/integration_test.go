package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/herrfennessey/brewmaster/api/internal/ai"
	"github.com/herrfennessey/brewmaster/api/internal/models"
	"github.com/herrfennessey/brewmaster/api/internal/router"
)

const sampleBeanText = `
Ethiopia Yirgacheffe Kochere — Dumerso Cooperative
Process: Washed
Altitude: 1800–2200m
Roast: Light
Roast date: 2024-01-15
Tasting notes: blueberry, jasmine, lemon zest
`

// integrationClient wraps a test server and HTTP client for making API requests.
type integrationClient struct {
	http    *http.Client
	baseURL string
}

func (c *integrationClient) post(t *testing.T, path string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	//nolint:gosec // test server URL is safe
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func (c *integrationClient) decode(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close() //nolint:errcheck
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func newIntegrationServer(t *testing.T) *integrationClient {
	t.Helper()
	provider, err := ai.NewOpenAIProvider()
	if err != nil {
		t.Fatalf("create AI provider: %v", err)
	}
	srv := httptest.NewServer(router.New(provider))
	t.Cleanup(srv.Close)
	return &integrationClient{baseURL: srv.URL, http: srv.Client()}
}

func TestIntegration(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set — skipping integration tests")
	}
	client := newIntegrationServer(t)
	t.Run("parse_bean_returns_valid_profile", func(t *testing.T) { testParseBeanReturnsValidProfile(t, client) })
	t.Run("generate_parameters_returns_valid_brew_params", func(t *testing.T) { testGenerateParametersReturnsValidBrewParams(t, client) })
}

func testParseBeanReturnsValidProfile(t *testing.T, client *integrationClient) {
	t.Helper()
	resp := client.post(t, "/api/parse-bean", map[string]string{
		"input_type": "text",
		"content":    sampleBeanText,
	})

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var profile models.BeanProfile
	client.decode(t, resp, &profile)

	if profile.ID == "" {
		t.Error("expected non-empty id")
	}
	if profile.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if !isValidConfidenceLevel(profile.Confidence.Level) {
		t.Errorf("invalid confidence level %q", profile.Confidence.Level)
	}
	// Yirgacheffe is unambiguous — expect at least country to be parsed.
	if profile.Parsed.OriginCountry == nil || *profile.Parsed.OriginCountry == "" {
		t.Error("expected origin_country to be extracted")
	}
	if profile.Parsed.Process == nil || *profile.Parsed.Process == "" {
		t.Error("expected process to be extracted")
	}
}

func testGenerateParametersReturnsValidBrewParams(t *testing.T, client *integrationClient) {
	t.Helper()
	parseResp := client.post(t, "/api/parse-bean", map[string]string{
		"input_type": "text",
		"content":    sampleBeanText,
	})
	if parseResp.StatusCode != http.StatusOK {
		t.Fatalf("parse-bean failed with status %d", parseResp.StatusCode)
	}
	var profile models.BeanProfile
	client.decode(t, parseResp, &profile)

	brewResp := client.post(t, "/api/generate-parameters", map[string]any{
		"bean_profile": profile,
		"target_drink": "espresso",
	})
	if brewResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(brewResp.Body)
		t.Fatalf("expected 200, got %d: %s", brewResp.StatusCode, body)
	}

	var params models.BrewParameters
	client.decode(t, brewResp, &params)

	if params.BeanID != profile.ID {
		t.Errorf("bean_id mismatch: got %q, want %q", params.BeanID, profile.ID)
	}
	assertParamValue(t, "dose_g", params.Parameters.DoseG)
	assertParamValue(t, "yield_g", params.Parameters.YieldG)
	assertParamValue(t, "temp_c", params.Parameters.TempC)
	assertParamValue(t, "time_s", params.Parameters.TimeS)
	assertParamValue(t, "preinfusion_s", params.Parameters.PreinfusionS)
	if !strings.HasPrefix(params.Parameters.Ratio, "1:") {
		t.Errorf("expected ratio like 1:X, got %q", params.Parameters.Ratio)
	}
	if params.Reasoning == "" {
		t.Error("expected non-empty reasoning")
	}
	if !isValidConfidenceLevel(params.Confidence.Level) {
		t.Errorf("invalid confidence level %q", params.Confidence.Level)
	}
	if params.Iteration < 1 {
		t.Errorf("expected iteration >= 1, got %d", params.Iteration)
	}
}

func TestIntegrationErrors(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set — skipping integration tests")
	}

	client := newIntegrationServer(t)

	t.Run("parse_bean_rejects_unsupported_input_type", func(t *testing.T) {
		resp := client.post(t, "/api/parse-bean", map[string]string{
			"input_type": "image",
			"content":    "some content",
		})
		defer resp.Body.Close() //nolint:errcheck
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("parse_bean_rejects_empty_content", func(t *testing.T) {
		resp := client.post(t, "/api/parse-bean", map[string]string{
			"input_type": "text",
			"content":    "",
		})
		defer resp.Body.Close() //nolint:errcheck
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("generate_parameters_rejects_missing_bean_id", func(t *testing.T) {
		resp := client.post(t, "/api/generate-parameters", map[string]any{
			"bean_profile": map[string]any{},
			"target_drink": "espresso",
		})
		defer resp.Body.Close() //nolint:errcheck
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})
}

func isValidConfidenceLevel(level string) bool {
	switch level {
	case "high", "medium", "low":
		return true
	}
	return false
}

func assertParamValue(t *testing.T, name string, p models.ParameterValue) {
	t.Helper()
	if p.Value <= 0 {
		t.Errorf("%s: expected value > 0, got %v", name, p.Value)
	}
	if p.Range[0] <= 0 || p.Range[1] <= 0 {
		t.Errorf("%s: expected positive range, got %v", name, p.Range)
	}
	if p.Range[0] > p.Range[1] {
		t.Errorf("%s: range min %v > max %v", name, p.Range[0], p.Range[1])
	}
}
