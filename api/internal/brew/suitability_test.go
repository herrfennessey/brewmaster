package brew_test

import (
	"testing"

	"github.com/herrfennessey/brewmaster/api/internal/brew"
)

func TestSuitability_GeshaLatte_Poor(t *testing.T) {
	bean := brew.CanonicalBean{Varietal: "gesha"}
	r := brew.ComputeSuitability(&bean, "latte")
	assertSuitability(t, r, "poor", brew.RuleGeshaMilk)
}

func TestSuitability_GeshaFlatWhite_Poor(t *testing.T) {
	bean := brew.CanonicalBean{Varietal: "gesha"}
	r := brew.ComputeSuitability(&bean, "flat white")
	assertSuitability(t, r, "poor", brew.RuleGeshaMilk)
}

func TestSuitability_AnaerobicFlatWhite_Poor(t *testing.T) {
	bean := brew.CanonicalBean{Process: "anaerobic"}
	r := brew.ComputeSuitability(&bean, "flat white")
	assertSuitability(t, r, "poor", brew.RuleAnaerobicMedMilk)
}

func TestSuitability_AnaerobicLatte_Poor(t *testing.T) {
	bean := brew.CanonicalBean{Process: "anaerobic"}
	r := brew.ComputeSuitability(&bean, "latte")
	assertSuitability(t, r, "poor", brew.RuleAnaerobicLatte)
}

func TestSuitability_LightLatte_Poor(t *testing.T) {
	bean := brew.CanonicalBean{RoastLevel: "light"}
	r := brew.ComputeSuitability(&bean, "latte")
	assertSuitability(t, r, "poor", brew.RuleLightLatte)
}

func TestSuitability_DarkPouroverBlack_Poor(t *testing.T) {
	bean := brew.CanonicalBean{RoastLevel: "dark"}
	r := brew.ComputeSuitability(&bean, "black")
	assertSuitability(t, r, "poor", brew.RuleDarkPourover)
}

func TestSuitability_LightFlatWhite_Suboptimal(t *testing.T) {
	bean := brew.CanonicalBean{RoastLevel: "light"}
	r := brew.ComputeSuitability(&bean, "flat white")
	assertSuitability(t, r, "suboptimal", brew.RuleLightMedMilk)
}

func TestSuitability_EthiopianWashedLatte_Suboptimal(t *testing.T) {
	bean := brew.CanonicalBean{OriginCountry: "ethiopia", Process: "washed"}
	r := brew.ComputeSuitability(&bean, "latte")
	assertSuitability(t, r, "suboptimal", brew.RuleEastAfricanLatte)
}

func TestSuitability_EthiopianNaturalLatte_Suboptimal(t *testing.T) {
	bean := brew.CanonicalBean{OriginCountry: "ethiopia", Process: "natural"}
	r := brew.ComputeSuitability(&bean, "latte")
	assertSuitability(t, r, "suboptimal", brew.RuleEastAfricanLatte)
}

func TestSuitability_BrazilianNaturalLatte_Suitable(t *testing.T) {
	bean := brew.CanonicalBean{OriginCountry: "brazil", Process: "natural", RoastLevel: "medium"}
	r := brew.ComputeSuitability(&bean, "latte")
	if r.Level != "suitable" {
		t.Errorf("expected suitable, got %q (rule %q)", r.Level, r.Rule)
	}
}

func TestSuitability_BrazilianNaturalDarkLatte_Ideal(t *testing.T) {
	bean := brew.CanonicalBean{OriginCountry: "brazil", Process: "natural", RoastLevel: "dark"}
	r := brew.ComputeSuitability(&bean, "latte")
	assertSuitability(t, r, "ideal", brew.RuleMilkOriginDarkRoast)
}

func TestSuitability_DarkCafeAuLait_Suboptimal(t *testing.T) {
	bean := brew.CanonicalBean{RoastLevel: "dark"}
	r := brew.ComputeSuitability(&bean, "cafe au lait")
	assertSuitability(t, r, "suboptimal", brew.RuleDarkCafeAuLait)
}

func TestSuitability_ColombianMediumWashedFlatWhite_Suitable(t *testing.T) {
	bean := brew.CanonicalBean{OriginCountry: "colombia", Process: "washed", RoastLevel: "medium"}
	r := brew.ComputeSuitability(&bean, "flat white")
	if r.Level != "suitable" {
		t.Errorf("expected suitable, got %q (rule %q)", r.Level, r.Rule)
	}
}

func TestSuitability_WashedEthiopianEspresso_Ideal(t *testing.T) {
	bean := brew.CanonicalBean{OriginCountry: "ethiopia", Process: "washed", RoastLevel: "medium"}
	r := brew.ComputeSuitability(&bean, "espresso")
	assertSuitability(t, r, "ideal", brew.RuleWashedEastAfricanBlack)
}

func TestSuitability_WashedMediumEspresso_Ideal(t *testing.T) {
	bean := brew.CanonicalBean{OriginCountry: "colombia", Process: "washed", RoastLevel: "medium"}
	r := brew.ComputeSuitability(&bean, "espresso")
	assertSuitability(t, r, "ideal", brew.RuleWashedLightBlack)
}

func TestSuitability_HoneyMediumLatte_Ideal(t *testing.T) {
	bean := brew.CanonicalBean{Process: "honey", RoastLevel: "medium"}
	r := brew.ComputeSuitability(&bean, "latte")
	assertSuitability(t, r, "ideal", brew.RuleHoneyMediumMilk)
}

func TestSuitability_WetHulledFlatWhite_Ideal(t *testing.T) {
	bean := brew.CanonicalBean{Process: "wet-hulled", RoastLevel: "medium"}
	r := brew.ComputeSuitability(&bean, "flat white")
	assertSuitability(t, r, "ideal", brew.RuleWetHulledMilk)
}

func TestSuitability_NaturalCortado_Suboptimal(t *testing.T) {
	bean := brew.CanonicalBean{Process: "natural"}
	r := brew.ComputeSuitability(&bean, "cortado")
	assertSuitability(t, r, "suboptimal", brew.RuleNaturalAnaeroCortado)
}

func TestSuitability_Suitable_HasNonEmptyReason(t *testing.T) {
	bean := brew.CanonicalBean{Process: "washed", RoastLevel: "medium"}
	r := brew.ComputeSuitability(&bean, "americano")
	if r.Level != "suitable" {
		t.Errorf("expected suitable, got %q", r.Level)
	}
	if r.Reason == "" {
		t.Error("expected non-empty reason for suitable pairing")
	}
}

func assertSuitability(t *testing.T, r brew.SuitabilityResult, wantLevel string, wantRule brew.RuleID) {
	t.Helper()
	if r.Level != wantLevel {
		t.Errorf("level: want %q, got %q", wantLevel, r.Level)
	}
	if r.Rule != wantRule {
		t.Errorf("rule: want %q, got %q", wantRule, r.Rule)
	}
	if r.Reason == "" {
		t.Error("reason must not be empty")
	}
}
