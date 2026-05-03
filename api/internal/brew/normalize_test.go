package brew_test

import (
	"testing"

	"github.com/herrfennessey/brewmaster/api/internal/brew"
	"github.com/herrfennessey/brewmaster/api/internal/models"
)

func strPtr(s string) *string   { return &s }
func f64Ptr(f float64) *float64 { return &f }

func TestNormalize_Varietal(t *testing.T) {
	bean := models.ParsedBean{Varietal: strPtr("Geisha")}
	cb := brew.Normalize(&bean)
	if cb.Varietal != "gesha" {
		t.Errorf("expected gesha, got %q", cb.Varietal)
	}
}

func TestNormalize_VarietalLowercase(t *testing.T) {
	bean := models.ParsedBean{Varietal: strPtr("BOURBON")}
	cb := brew.Normalize(&bean)
	if cb.Varietal != "bourbon" {
		t.Errorf("expected bourbon, got %q", cb.Varietal)
	}
}

func TestNormalize_ProcessGilingBasah(t *testing.T) {
	bean := models.ParsedBean{Process: strPtr("giling basah")}
	cb := brew.Normalize(&bean)
	if cb.Process != "wet-hulled" {
		t.Errorf("expected wet-hulled, got %q", cb.Process)
	}
}

func TestNormalize_ProcessPreserved(t *testing.T) {
	bean := models.ParsedBean{Process: strPtr("Washed")}
	cb := brew.Normalize(&bean)
	if cb.Process != "washed" {
		t.Errorf("expected washed, got %q", cb.Process)
	}
}

func TestNormalize_OriginCountryLowercase(t *testing.T) {
	bean := models.ParsedBean{OriginCountry: strPtr("ETHIOPIA")}
	cb := brew.Normalize(&bean)
	if cb.OriginCountry != "ethiopia" {
		t.Errorf("expected ethiopia, got %q", cb.OriginCountry)
	}
}

func TestNormalize_RoastNormalization(t *testing.T) {
	tests := []struct{ input, want string }{
		{"Light", "light"},
		{"Medium-Light", "light"},
		{"medium", "medium"},
		{"Dark", "dark"},
		{"Medium-Dark", "dark"},
		{"unknown roast", "medium"},
	}
	for _, tt := range tests {
		bean := models.ParsedBean{RoastLevel: strPtr(tt.input)}
		cb := brew.Normalize(&bean)
		if cb.RoastLevel != tt.want {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.want, cb.RoastLevel)
		}
	}
}

func TestNormalize_AltitudeKnown(t *testing.T) {
	bean := models.ParsedBean{AltitudeM: f64Ptr(1800)}
	cb := brew.Normalize(&bean)
	if !cb.AltitudeKnown || cb.AltitudeM != 1800 {
		t.Errorf("expected altitude known=true, 1800, got known=%v, %.0f", cb.AltitudeKnown, cb.AltitudeM)
	}
}

func TestNormalize_AltitudeUnknown(t *testing.T) {
	bean := models.ParsedBean{}
	cb := brew.Normalize(&bean)
	if cb.AltitudeKnown {
		t.Error("expected altitude unknown")
	}
}

func TestNormalizeDrink_CafeAuLait(t *testing.T) {
	got := brew.NormalizeDrink("café au lait")
	if got != "cafe au lait" {
		t.Errorf("expected \"cafe au lait\", got %q", got)
	}
}

func TestNormalizeDrink_Lowercase(t *testing.T) {
	got := brew.NormalizeDrink("Espresso")
	if got != "espresso" {
		t.Errorf("expected espresso, got %q", got)
	}
}
