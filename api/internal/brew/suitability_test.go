package brew_test

import (
	"strings"
	"testing"

	"github.com/herrfennessey/brewmaster/api/internal/brew"
)

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

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

func TestSuitability_EthiopianMediumLightLatte_Suboptimal(t *testing.T) {
	// Light roast + latte is already "poor" (RuleLightLatte) — that rule fires
	// first. Medium-light Ethiopia is the band where the east-african rule
	// adds its signal: clear origin character still gets buried by milk.
	bean := brew.CanonicalBean{OriginCountry: "ethiopia", Process: "washed", RoastLevel: "medium-light"}
	r := brew.ComputeSuitability(&bean, "latte")
	assertSuitability(t, r, "suboptimal", brew.RuleEastAfricanLatte)
}

func TestSuitability_EthiopianMediumNaturalLatte_Suboptimal(t *testing.T) {
	// Medium-roasted Ethiopian + latte is still suboptimal — origin character is
	// front-and-center at medium and gets buried by milk.
	bean := brew.CanonicalBean{OriginCountry: "ethiopia", Process: "natural", RoastLevel: "medium"}
	r := brew.ComputeSuitability(&bean, "latte")
	assertSuitability(t, r, "suboptimal", brew.RuleEastAfricanLatte)
}

func TestSuitability_EthiopianDarkLatte_Suitable(t *testing.T) {
	// Dark-roasted Ethiopian Sidamo in a latte is a perfectly classic pairing —
	// the v2 rule no longer flags it.
	bean := brew.CanonicalBean{OriginCountry: "ethiopia", Process: "natural", RoastLevel: "dark"}
	r := brew.ComputeSuitability(&bean, "latte")
	if r.Level != "suitable" && r.Level != "ideal" {
		t.Errorf("expected suitable or ideal for dark Ethiopian latte, got %q (rule %q)", r.Level, r.Rule)
	}
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

func TestSuitability_AnaerobicCortado_Suboptimal(t *testing.T) {
	// v2: anaerobic-only — naturals no longer fire this rule.
	bean := brew.CanonicalBean{Process: "anaerobic"}
	r := brew.ComputeSuitability(&bean, "cortado")
	assertSuitability(t, r, "suboptimal", brew.RuleNaturalAnaeroCortado)
}

func TestSuitability_NaturalCortado_Suitable(t *testing.T) {
	// Plain natural in a cortado is fine in v2 (was suboptimal in v1).
	bean := brew.CanonicalBean{Process: "natural", RoastLevel: "medium"}
	r := brew.ComputeSuitability(&bean, "cortado")
	if r.Level != "suitable" {
		t.Errorf("expected suitable for medium natural cortado, got %q (rule %q)", r.Level, r.Rule)
	}
}

func TestSuitability_WashedSL28Latte_Suboptimal(t *testing.T) {
	// SL28 washed + milk → sharp acidity does not integrate.
	bean := brew.CanonicalBean{Varietal: "sl28", Process: "washed", RoastLevel: "medium"}
	r := brew.ComputeSuitability(&bean, "latte")
	assertSuitability(t, r, "suboptimal", brew.RuleSL28MedHighMilk)
}

func TestSuitability_HoneySL28Latte_Suitable(t *testing.T) {
	// Honey-processed SL28 integrates with milk fine — v2 narrows the rule to
	// washed only.
	bean := brew.CanonicalBean{Varietal: "sl28", Process: "honey", RoastLevel: "medium"}
	r := brew.ComputeSuitability(&bean, "latte")
	if r.Level == "suboptimal" && r.Rule == brew.RuleSL28MedHighMilk {
		t.Errorf("honey SL28 latte should not trigger washed-SL28 rule (got %q)", r.Rule)
	}
}

func TestSuitability_DarkPourover_ReasonMentionsExtraction(t *testing.T) {
	// v2 reason rewrites "espresso pressure balances bitterness" (physically
	// dubious) to call out long contact time and over-extraction.
	bean := brew.CanonicalBean{RoastLevel: "dark"}
	r := brew.ComputeSuitability(&bean, "black")
	if r.Rule != brew.RuleDarkPourover {
		t.Fatalf("expected RuleDarkPourover, got %q", r.Rule)
	}
	if !contains(r.Reason, "over-extract") && !contains(r.Reason, "contact time") {
		t.Errorf("reason should mention over-extraction or contact time; got %q", r.Reason)
	}
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
