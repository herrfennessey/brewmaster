package brew_test

import (
	"testing"
	"time"

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
		{"Medium-Light", "medium-light"},
		{"medium light", "medium-light"},
		{"medium", "medium"},
		{"Dark", "dark"},
		{"Medium-Dark", "dark"},
		{"unknown roast", ""},
		{"", ""},
	}
	for _, tt := range tests {
		bean := models.ParsedBean{RoastLevel: strPtr(tt.input)}
		cb := brew.Normalize(&bean)
		if cb.RoastLevel != tt.want {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.want, cb.RoastLevel)
		}
	}
}

func TestNormalize_RoastNil_PreservesUnknown(t *testing.T) {
	bean := models.ParsedBean{} // RoastLevel is nil
	cb := brew.Normalize(&bean)
	if cb.RoastLevel != "" {
		t.Errorf("expected empty (unknown) roast level for nil source, got %q", cb.RoastLevel)
	}
}

func TestNormalize_VarietalAliases(t *testing.T) {
	tests := []struct{ input, want string }{
		{"SL-28", "sl28"},
		{"sl 28", "sl28"},
		{"SL-34", "sl34"},
		{"Geisha", "gesha"},
		{"BOURBON", "bourbon"},
	}
	for _, tt := range tests {
		bean := models.ParsedBean{Varietal: strPtr(tt.input)}
		cb := brew.Normalize(&bean)
		if cb.Varietal != tt.want {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.want, cb.Varietal)
		}
	}
}

func TestNormalize_RoastDate_Parsed(t *testing.T) {
	bean := models.ParsedBean{RoastDate: strPtr("2026-04-01")}
	cb := brew.Normalize(&bean)
	if cb.RoastDate == nil {
		t.Fatal("expected RoastDate to be parsed, got nil")
	}
	want := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !cb.RoastDate.Equal(want) {
		t.Errorf("expected %v, got %v", want, *cb.RoastDate)
	}
}

func TestNormalize_RoastDate_InvalidIgnored(t *testing.T) {
	bean := models.ParsedBean{RoastDate: strPtr("not-a-date")}
	cb := brew.Normalize(&bean)
	if cb.RoastDate != nil {
		t.Errorf("expected RoastDate to be nil for invalid input, got %v", cb.RoastDate)
	}
}

func TestNormalize_DaysSinceRoast(t *testing.T) {
	roastDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	bean := brew.CanonicalBean{RoastDate: &roastDate}
	days, known := brew.DaysSinceRoast(&bean, now)
	if !known {
		t.Fatal("expected days to be known")
	}
	if days != 30 {
		t.Errorf("expected 30 days, got %d", days)
	}
}

func TestNormalize_DaysSinceRoast_Unknown(t *testing.T) {
	bean := brew.CanonicalBean{}
	_, known := brew.DaysSinceRoast(&bean, time.Now())
	if known {
		t.Error("expected days to be unknown when RoastDate is nil")
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
