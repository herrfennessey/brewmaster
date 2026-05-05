package brew_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/herrfennessey/brewmaster/api/internal/brew"
)

// fixedNow is a stable reference timestamp for deterministic tests. Choose a
// date such that beans without an explicit roast_date neither trip "fresh"
// nor "stale" nudges (the default fallback path is no roast date known).
var fixedNow = time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)

func compute(bean brew.CanonicalBean, method, drink string) brew.ComputedParams {
	return brew.ComputeParams(&bean, method, drink, fixedNow)
}

func TestCalc_Espresso_LightWashed_HighAltitude(t *testing.T) {
	// Light washed @ 1900m → roast-base 94°C + altitude(+1.0) = 95°C, ratio 2.4.
	bean := brew.CanonicalBean{Process: "washed", RoastLevel: "light", AltitudeKnown: true, AltitudeM: 1900}
	cp := compute(bean, "espresso", "espresso")

	assertApprox(t, "temp", cp.Params.TempC.Value, 95.0)
	assertRatio(t, cp.Params.Ratio, "1:2.4")
	assertRulesContain(t, cp.AppliedRules, brew.RuleHighAltitudeTemp, brew.RuleLightRoastHighTemp)
}

func TestCalc_Espresso_NaturalDarkUnknownAlt(t *testing.T) {
	// Natural dark, unknown alt → roast-base 90°C + natural(0) = 90°C.
	// Ratio: 2.2 + natural(-0.2) + dark(-0.2) = 1.8.
	bean := brew.CanonicalBean{Process: "natural", RoastLevel: "dark"}
	cp := compute(bean, "espresso", "espresso")

	assertApprox(t, "temp", cp.Params.TempC.Value, 90.0)
	assertRatio(t, cp.Params.Ratio, "1:1.8")
	assertRulesContain(t, cp.AppliedRules, brew.RuleNaturalLowerExtract, brew.RuleDarkRoastLowTemp)
}

func TestCalc_Espresso_MediumWashedLatte(t *testing.T) {
	// Medium washed, unknown alt, latte → 92°C base; ratio 2.2 + latte(0.3) = 2.5.
	bean := brew.CanonicalBean{Process: "washed", RoastLevel: "medium"}
	cp := compute(bean, "espresso", "latte")

	assertApprox(t, "temp", cp.Params.TempC.Value, 92.0)
	assertRatio(t, cp.Params.Ratio, "1:2.5")
	assertRulesContain(t, cp.AppliedRules, brew.RuleDrinkRatioAdjust)
}

func TestCalc_Espresso_UnknownRoast_DefaultsToMediumLight(t *testing.T) {
	// No roast level → medium-light default = 93°C; rule surfaced for traceability.
	bean := brew.CanonicalBean{Process: "washed"}
	cp := compute(bean, "espresso", "espresso")

	assertApprox(t, "temp", cp.Params.TempC.Value, 93.0)
	assertRulesContain(t, cp.AppliedRules, brew.RuleUnknownRoastMediumLightDefault)
}

func TestCalc_Espresso_MediumLightExplicit_NotMediumLightDefault(t *testing.T) {
	// Explicit medium-light should NOT trigger the unknown-default rule.
	bean := brew.CanonicalBean{RoastLevel: "medium-light"}
	cp := compute(bean, "espresso", "espresso")

	assertApprox(t, "temp", cp.Params.TempC.Value, 93.0)
	for _, r := range cp.AppliedRules {
		if r == brew.RuleUnknownRoastMediumLightDefault {
			t.Errorf("explicit medium-light should not surface unknown-default rule")
		}
	}
}

func TestCalc_Espresso_VarietalGesha_CoolerByHalfDegree(t *testing.T) {
	// Light gesha @ 1500m → 94 + 0.5(alt) - 0.5(soft varietal) = 94.0°C.
	bean := brew.CanonicalBean{RoastLevel: "light", Varietal: "gesha", AltitudeKnown: true, AltitudeM: 1500}
	cp := compute(bean, "espresso", "espresso")

	assertApprox(t, "temp", cp.Params.TempC.Value, 94.0)
	assertRulesContain(t, cp.AppliedRules, brew.RuleVarietalSoft)
}

func TestCalc_Espresso_VarietalRobusta_HotterByOneDegree(t *testing.T) {
	// Medium robusta, unknown alt → 92 + 1(hard varietal) = 93°C.
	bean := brew.CanonicalBean{RoastLevel: "medium", Varietal: "robusta"}
	cp := compute(bean, "espresso", "espresso")

	assertApprox(t, "temp", cp.Params.TempC.Value, 93.0)
	assertRulesContain(t, cp.AppliedRules, brew.RuleVarietalHard)
}

func TestCalc_Espresso_DarkWetHulledLowAlt_ClampedToMin(t *testing.T) {
	// Dark wet-hulled @ 800m → 90 + altitude(-0.5) + wet-hulled(-1) = 88.5°C.
	// This was the worst pathology of v1 (would land at 85°C); confirm it now
	// stays inside the operating envelope.
	bean := brew.CanonicalBean{RoastLevel: "dark", Process: "wet-hulled", AltitudeKnown: true, AltitudeM: 800}
	cp := compute(bean, "espresso", "espresso")

	if cp.Params.TempC.Value < 86.0 {
		t.Errorf("temp %.1f below floor; clamp failed", cp.Params.TempC.Value)
	}
	assertApprox(t, "temp", cp.Params.TempC.Value, 88.5)
}

func TestCalc_Espresso_StaleBean_GetsBoost(t *testing.T) {
	// Roasted 90 days ago → stale: +1°C, +2s time.
	roastDate := fixedNow.AddDate(0, 0, -90)
	bean := brew.CanonicalBean{RoastLevel: "medium", RoastDate: &roastDate}
	cp := compute(bean, "espresso", "espresso")

	assertApprox(t, "temp", cp.Params.TempC.Value, 93.0) // 92 + 1
	assertApprox(t, "time", cp.Params.TimeS.Value, 32.0) // 30 + 2
	assertRulesContain(t, cp.AppliedRules, brew.RuleRoastStale)
}

func TestCalc_Espresso_FreshBean_LongerPreinfusion(t *testing.T) {
	// Roasted 2 days ago → very fresh: extended preinfusion, +1s time, no temp shift.
	roastDate := fixedNow.AddDate(0, 0, -2)
	bean := brew.CanonicalBean{RoastLevel: "medium", RoastDate: &roastDate}
	cp := compute(bean, "espresso", "espresso")

	assertApprox(t, "temp", cp.Params.TempC.Value, 92.0)
	assertApprox(t, "time", cp.Params.TimeS.Value, 31.0)
	assertApprox(t, "preinfusion", cp.Params.PreinfusionS.Value, 8.0) // 6 + 2
	assertRulesContain(t, cp.AppliedRules, brew.RuleRoastFresh)
}

func TestCalc_Espresso_PostPeak_FlagOnly(t *testing.T) {
	// Roasted 45 days ago → post-peak: rule fires but no numeric change.
	roastDate := fixedNow.AddDate(0, 0, -45)
	bean := brew.CanonicalBean{RoastLevel: "medium", RoastDate: &roastDate}
	cp := compute(bean, "espresso", "espresso")

	assertApprox(t, "temp", cp.Params.TempC.Value, 92.0)
	assertApprox(t, "time", cp.Params.TimeS.Value, 30.0)
	assertRulesContain(t, cp.AppliedRules, brew.RuleRoastPostPeak)
}

func TestCalc_Espresso_YieldEqualsDoseTimesRatio(t *testing.T) {
	bean := brew.CanonicalBean{Process: "washed", RoastLevel: "medium"}
	cp := compute(bean, "espresso", "espresso")

	var ratioNum float64
	_, err := fmt.Sscanf(cp.Params.Ratio, "1:%f", &ratioNum)
	if err != nil {
		t.Fatalf("failed to parse ratio %q: %v", cp.Params.Ratio, err)
	}
	expected := cp.Params.DoseG.Value * ratioNum
	if abs(cp.Params.YieldG.Value-expected) > 0.5 {
		t.Errorf("yield %.1f != dose %.1f × ratio %.2f (= %.1f)", cp.Params.YieldG.Value, cp.Params.DoseG.Value, ratioNum, expected)
	}
}

func TestCalc_Espresso_Version(t *testing.T) {
	bean := brew.CanonicalBean{}
	cp := compute(bean, "espresso", "espresso")
	if cp.Version != brew.RulesetVersion {
		t.Errorf("version: want %q, got %q", brew.RulesetVersion, cp.Version)
	}
}

func TestCalc_Espresso_RangesValid(t *testing.T) {
	bean := brew.CanonicalBean{Process: "honey", RoastLevel: "light", AltitudeKnown: true, AltitudeM: 1200}
	cp := compute(bean, "espresso", "cappuccino")

	p := cp.Params
	assertRange(t, "temp", p.TempC.Value, p.TempC.Range[0], p.TempC.Range[1])
	assertRange(t, "yield", p.YieldG.Value, p.YieldG.Range[0], p.YieldG.Range[1])
	assertRange(t, "time", p.TimeS.Value, p.TimeS.Range[0], p.TimeS.Range[1])
	assertRange(t, "preinfusion", p.PreinfusionS.Value, p.PreinfusionS.Range[0], p.PreinfusionS.Range[1])
}

func TestCalc_Pourover_MediumWashed(t *testing.T) {
	// Medium washed → ratio 15.5, water = 15 × 15.5 = 232.5g, bloom 35s, temp 93°C.
	bean := brew.CanonicalBean{Process: "washed", RoastLevel: "medium"}
	cp := compute(bean, "pourover", "black")

	assertApprox(t, "water", cp.Params.YieldG.Value, 232.5)
	assertApprox(t, "bloom", cp.Params.PreinfusionS.Value, 35.0)
	assertRatio(t, cp.Params.Ratio, "1:15.5")
	assertApprox(t, "temp", cp.Params.TempC.Value, 93.0)
}

func TestCalc_Pourover_LightRoastHigherTemp(t *testing.T) {
	bean := brew.CanonicalBean{RoastLevel: "light"}
	cp := compute(bean, "pourover", "black")
	// Pourover light base 95°C, no altitude/varietal/process/freshness deltas.
	assertApprox(t, "temp", cp.Params.TempC.Value, 95.0)
}

func TestCalc_Pourover_RatioString(t *testing.T) {
	bean := brew.CanonicalBean{RoastLevel: "medium", Process: "natural"}
	cp := compute(bean, "pourover", "black")
	// Medium base ratio 15.5 + natural(-0.5) = 15.0.
	assertRatio(t, cp.Params.Ratio, "1:15.0")
}

func TestCalc_FallbackReasoning_Empty(t *testing.T) {
	s := brew.FallbackReasoning(nil)
	if s == "" {
		t.Error("fallback reasoning should return a non-empty string")
	}
}

func TestCalc_FallbackReasoning_WithRules(t *testing.T) {
	s := brew.FallbackReasoning([]brew.RuleID{brew.RuleHighAltitudeTemp, brew.RuleLightRoastHighTemp})
	if !strings.Contains(s, "altitude") {
		t.Errorf("expected altitude mention, got %q", s)
	}
	if s[0:1] != strings.ToUpper(s[0:1]) {
		t.Error("first character should be uppercase")
	}
}

func TestCalc_FallbackReasoning_NewRulesHavePhrases(t *testing.T) {
	// Sanity-check that all rules attachable in v2 have a fallback phrase, so
	// the deterministic reasoning string is meaningful when the LLM annotation
	// fails.
	newRules := []brew.RuleID{
		brew.RuleVarietalSoft,
		brew.RuleVarietalHard,
		brew.RuleRoastFresh,
		brew.RuleRoastStale,
		brew.RuleRoastPostPeak,
		brew.RuleUnknownRoastMediumLightDefault,
	}
	for _, r := range newRules {
		s := brew.FallbackReasoning([]brew.RuleID{r})
		if s == "" || s == "Standard parameters for this bean and extraction method." {
			t.Errorf("rule %q has no fallback phrase", r)
		}
	}
}

func assertApprox(t *testing.T, name string, got, want float64) {
	t.Helper()
	if abs(got-want) > 0.1 {
		t.Errorf("%s: want %.1f, got %.1f", name, want, got)
	}
}

func assertRatio(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("ratio: want %q, got %q", want, got)
	}
}

func assertRulesContain(t *testing.T, rules []brew.RuleID, want ...brew.RuleID) {
	t.Helper()
	for _, w := range want {
		found := false
		for _, r := range rules {
			if r == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected rule %q in applied rules %v", w, rules)
		}
	}
}

func assertRange(t *testing.T, name string, val, lo, hi float64) {
	t.Helper()
	if val < lo || val > hi {
		t.Errorf("%s value %.1f outside range [%.1f, %.1f]", name, val, lo, hi)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
