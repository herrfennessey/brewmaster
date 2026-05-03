package brew_test

import (
	"testing"

	"github.com/herrfennessey/brewmaster/api/internal/brew"
)

func TestConfidence_AllPresent_High(t *testing.T) {
	bean := brew.CanonicalBean{
		RoastLevel:    "medium",
		Process:       "washed",
		AltitudeKnown: true,
		AltitudeM:     1500,
		OriginCountry: "ethiopia",
		OriginRegion:  "yirgacheffe",
		Varietal:      "heirloom",
		FlavorNotes:   []string{"blueberry"},
	}
	c := brew.ComputeConfidence(&bean)
	if c.Level != "high" {
		t.Errorf("expected high, got %q", c.Level)
	}
}

func TestConfidence_MissingRoastLevel_Medium(t *testing.T) {
	// Missing roast_level (weight 5) → score 15 → medium
	bean := brew.CanonicalBean{
		Process:       "washed",
		AltitudeKnown: true,
		AltitudeM:     1500,
		OriginCountry: "ethiopia",
		OriginRegion:  "yirgacheffe",
		Varietal:      "heirloom",
		FlavorNotes:   []string{"citrus"},
	}
	c := brew.ComputeConfidence(&bean)
	if c.Level != "medium" {
		t.Errorf("expected medium, got %q", c.Level)
	}
}

func TestConfidence_MissingRoastProcessAltitude_Low(t *testing.T) {
	// Missing roast(5) + process(4) + altitude(4) = -13 → score 7 → low
	bean := brew.CanonicalBean{
		OriginCountry: "ethiopia",
		OriginRegion:  "sidama",
		Varietal:      "heirloom",
		FlavorNotes:   []string{"floral"},
	}
	c := brew.ComputeConfidence(&bean)
	if c.Level != "low" {
		t.Errorf("expected low, got %q", c.Level)
	}
}

func TestConfidence_MissingOnlyLowWeightFields_High(t *testing.T) {
	// Missing only origin_region(1) and flavor_notes(1) → score 18 → high
	bean := brew.CanonicalBean{
		RoastLevel:    "medium",
		Process:       "washed",
		AltitudeKnown: true,
		AltitudeM:     1500,
		OriginCountry: "colombia",
		Varietal:      "caturra",
	}
	c := brew.ComputeConfidence(&bean)
	if c.Level != "high" {
		t.Errorf("expected high, got %q (score should be 18)", c.Level)
	}
}

func TestConfidence_ReasonMentionsMissingFields(t *testing.T) {
	bean := brew.CanonicalBean{OriginCountry: "colombia"}
	c := brew.ComputeConfidence(&bean)
	if c.Reason == "" {
		t.Error("expected non-empty reason")
	}
	if c.Level == "high" {
		t.Errorf("expected medium or low for sparse bean, got %q", c.Level)
	}
}

func TestConfidence_CompleteBean_ReasonMentionsComplete(t *testing.T) {
	bean := brew.CanonicalBean{
		RoastLevel: "light", Process: "washed", AltitudeKnown: true,
		OriginCountry: "kenya", OriginRegion: "nyeri", Varietal: "sl28",
		FlavorNotes: []string{"blackcurrant"},
	}
	c := brew.ComputeConfidence(&bean)
	if c.Level != "high" {
		t.Errorf("expected high, got %q", c.Level)
	}
	if c.Reason == "" {
		t.Error("expected non-empty reason")
	}
}
