package brew_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/herrfennessey/brewmaster/api/internal/brew"
)

func TestCalc_Espresso_LightWashed_HighAltitude(t *testing.T) {
	// Light washed, 1900m → base 95°C + light +1°C = 96°C, ratio 2.2+0.2=2.4
	bean := brew.CanonicalBean{Process: "washed", RoastLevel: "light", AltitudeKnown: true, AltitudeM: 1900}
	cp := brew.ComputeParams(&bean, "espresso", "espresso")

	assertApprox(t, "temp", cp.Params.TempC.Value, 96.0)
	assertRatio(t, cp.Params.Ratio, "1:2.4")
	assertRulesContain(t, cp.AppliedRules, brew.RuleHighAltitudeTemp, brew.RuleLightRoastHighTemp)
}

func TestCalc_Espresso_NaturalDarkUnknownAlt(t *testing.T) {
	// Natural dark, unknown alt → base 92.5 + natural(-1) + dark(-2) = 89.5°C
	// ratio: 2.2 + natural(-0.2) + dark(-0.2) = 1.8
	bean := brew.CanonicalBean{Process: "natural", RoastLevel: "dark"}
	cp := brew.ComputeParams(&bean, "espresso", "espresso")

	assertApprox(t, "temp", cp.Params.TempC.Value, 89.5)
	assertRatio(t, cp.Params.Ratio, "1:1.8")
	assertRulesContain(t, cp.AppliedRules, brew.RuleNaturalLowerExtract, brew.RuleDarkRoastLowTemp)
}

func TestCalc_Espresso_MediumWashedLatte(t *testing.T) {
	// Nil altitude, medium washed, latte → base 92.5, ratio 2.2 + latte(0.3) = 2.5
	bean := brew.CanonicalBean{Process: "washed", RoastLevel: "medium"}
	cp := brew.ComputeParams(&bean, "espresso", "latte")

	assertApprox(t, "temp", cp.Params.TempC.Value, 92.5)
	assertRatio(t, cp.Params.Ratio, "1:2.5")
	assertRulesContain(t, cp.AppliedRules, brew.RuleDrinkRatioAdjust)
}

func TestCalc_Espresso_YieldEqualsDoseTimesRatio(t *testing.T) {
	bean := brew.CanonicalBean{Process: "washed", RoastLevel: "medium"}
	cp := brew.ComputeParams(&bean, "espresso", "espresso")

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
	cp := brew.ComputeParams(&bean, "espresso", "espresso")
	if cp.Version != brew.RulesetVersion {
		t.Errorf("version: want %q, got %q", brew.RulesetVersion, cp.Version)
	}
}

func TestCalc_Espresso_RangesValid(t *testing.T) {
	bean := brew.CanonicalBean{Process: "honey", RoastLevel: "light", AltitudeKnown: true, AltitudeM: 1200}
	cp := brew.ComputeParams(&bean, "espresso", "cappuccino")

	p := cp.Params
	assertRange(t, "temp", p.TempC.Value, p.TempC.Range[0], p.TempC.Range[1])
	assertRange(t, "yield", p.YieldG.Value, p.YieldG.Range[0], p.YieldG.Range[1])
	assertRange(t, "time", p.TimeS.Value, p.TimeS.Range[0], p.TimeS.Range[1])
	assertRange(t, "preinfusion", p.PreinfusionS.Value, p.PreinfusionS.Range[0], p.PreinfusionS.Range[1])
}

func TestCalc_Pourover_MediumWashed(t *testing.T) {
	// Medium washed → ratio 15.5, water = 15 × 15.5 = 232.5g, bloom 35s
	bean := brew.CanonicalBean{Process: "washed", RoastLevel: "medium"}
	cp := brew.ComputeParams(&bean, "pourover", "black")

	assertApprox(t, "water", cp.Params.YieldG.Value, 232.5)
	assertApprox(t, "bloom", cp.Params.PreinfusionS.Value, 35.0)
	assertRatio(t, cp.Params.Ratio, "1:15.5")
}

func TestCalc_Pourover_LightRoastHigherTemp(t *testing.T) {
	bean := brew.CanonicalBean{RoastLevel: "light"}
	cp := brew.ComputeParams(&bean, "pourover", "black")
	// Unknown altitude base 93.5 + light +1 = 94.5
	assertApprox(t, "temp", cp.Params.TempC.Value, 94.5)
}

func TestCalc_Pourover_RatioString(t *testing.T) {
	bean := brew.CanonicalBean{RoastLevel: "medium", Process: "natural"}
	cp := brew.ComputeParams(&bean, "pourover", "black")
	// medium base ratio 15.5 + natural(-0.5) = 15.0
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
