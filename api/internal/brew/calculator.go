package brew

import (
	"fmt"
	"math"
	"strings"

	"github.com/herrfennessey/brewmaster/api/internal/models"
)

// ComputedParams holds the deterministically computed brew parameters.
// Fields are ordered so pointer-containing fields (Version, AppliedRules) precede
// the large float64-heavy Params struct, minimizing the GC scan range.
type ComputedParams struct {
	Version      string
	AppliedRules []RuleID
	Params       models.BrewParams
}

// pouroverRoastAdj bundles the pourover roast adjustments to avoid a 6-value return.
type pouroverRoastAdj struct {
	Rule                                             RuleID
	DeltaTemp, RatioVal, RatioLo, RatioHi, DeltaTime float64
}

// ComputeParams deterministically calculates brew parameters for the given bean, method, and drink.
func ComputeParams(bean *CanonicalBean, method, drink string) ComputedParams {
	drink = NormalizeDrink(drink)
	if method == "pourover" {
		return computePourover(bean)
	}
	return computeEspresso(bean, drink)
}

// FallbackReasoning generates a reasoning string from applied parameter rules
// for use when the LLM annotation call fails.
func FallbackReasoning(rules []RuleID) string {
	//nolint:gosec // G101 false positive — these are descriptive phrases, not credentials.
	phrases := map[RuleID]string{
		RuleHighAltitudeTemp:    "high altitude suggests dense beans requiring higher brew temperature",
		RuleLightRoastHighTemp:  "light roast benefits from higher temperature and extended extraction",
		RuleDarkRoastLowTemp:    "dark roast uses lower temperature to avoid bitterness",
		RuleNaturalLowerExtract: "natural process requires reduced extraction to balance sweetness",
		RuleAnaerobicReduce:     "anaerobic process needs reduced extraction to preserve fermentation character",
		RuleHoneyAdjust:         "honey process calls for slightly reduced extraction",
		RuleWetHulledCooler:     "wet-hulled process uses lower temperature for body clarity",
		RuleDrinkRatioAdjust:    "drink style requires a higher yield ratio",
	}
	var parts []string
	for _, r := range rules {
		if p, ok := phrases[r]; ok {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return "Standard parameters for this bean and extraction method."
	}
	parts[0] = strings.ToUpper(parts[0][:1]) + parts[0][1:]
	return strings.Join(parts, "; ") + "."
}

func computeEspresso(bean *CanonicalBean, drink string) ComputedParams {
	const baseDose = 18.5
	const baseRatio = 2.2
	const baseTime = 30.0

	baseTemp, tempLo, tempHi, altRule := espressoTemp(bean)
	dTemp, dRatio, procRule := processEspressoAdj(bean.Process)
	rTemp, rRatio, rTime, roastRule := roastEspressoAdj(bean.RoastLevel)
	dDrink, drinkRule := drinkRatioDelta(drink)

	temp := baseTemp + dTemp + rTemp
	ratio := baseRatio + dRatio + rRatio + dDrink
	timeVal := baseTime + rTime
	yieldVal := baseDose * ratio

	return ComputedParams{
		Version:      RulesetVersion,
		AppliedRules: collectRules(altRule, procRule, roastRule, drinkRule),
		Params: models.BrewParams{
			DoseG:        pv(baseDose, 18.0, 20.0),
			YieldG:       pv(yieldVal, yieldVal-3, yieldVal+3),
			Ratio:        fmt.Sprintf("1:%.1f", ratio),
			TempC:        pv(temp, tempLo+dTemp+rTemp, tempHi+dTemp+rTemp),
			TimeS:        pv(timeVal, timeVal-4, timeVal+4),
			PreinfusionS: espressoPreinfusion(bean.RoastLevel),
		},
	}
}

func computePourover(bean *CanonicalBean) ComputedParams {
	const baseDose = 15.0
	const baseTime = 210.0
	const bloomS = 35.0

	baseTemp, tempLo, tempHi, altRule := pouroverTemp(bean)
	ra := roastPouroverAdj(bean.RoastLevel)
	dRatio, procRule := processPouroverAdj(bean.Process)

	temp := baseTemp + ra.DeltaTemp
	ratio := ra.RatioVal + dRatio
	ratioLo := ra.RatioLo + dRatio
	ratioHi := ra.RatioHi + dRatio
	timeVal := baseTime + ra.DeltaTime
	waterVal := baseDose * ratio

	return ComputedParams{
		Version:      RulesetVersion,
		AppliedRules: collectRules(altRule, procRule, ra.Rule),
		Params: models.BrewParams{
			DoseG:        pv(baseDose, 14.0, 17.0),
			YieldG:       pv(waterVal, baseDose*ratioLo, baseDose*ratioHi),
			Ratio:        fmt.Sprintf("1:%.1f", ratio),
			TempC:        pv(temp, tempLo+ra.DeltaTemp, tempHi+ra.DeltaTemp),
			TimeS:        pv(timeVal, timeVal-30, timeVal+30),
			PreinfusionS: pv(bloomS, 30.0, 45.0),
		},
	}
}

func espressoTemp(bean *CanonicalBean) (base, lo, hi float64, rule RuleID) {
	if !bean.AltitudeKnown {
		return 92.5, 91, 94, ""
	}
	switch {
	case bean.AltitudeM < 1000:
		return 89.0, 88, 90, ""
	case bean.AltitudeM < 1400:
		return 91.0, 90, 92, ""
	case bean.AltitudeM < 1800:
		return 93.0, 92, 94, ""
	default:
		return 95.0, 94, 96, RuleHighAltitudeTemp
	}
}

func pouroverTemp(bean *CanonicalBean) (base, lo, hi float64, rule RuleID) {
	if !bean.AltitudeKnown {
		return 93.5, 92, 95, ""
	}
	switch {
	case bean.AltitudeM < 1000:
		return 90.0, 89, 91, ""
	case bean.AltitudeM < 1400:
		return 92.0, 91, 93, ""
	case bean.AltitudeM < 1800:
		return 94.0, 93, 95, ""
	default:
		return 96.0, 95, 97, RuleHighAltitudeTemp
	}
}

func processEspressoAdj(process string) (deltaTemp, deltaRatio float64, rule RuleID) {
	switch process {
	case "natural":
		return -1.0, -0.2, RuleNaturalLowerExtract
	case "honey":
		return -0.5, -0.1, RuleHoneyAdjust
	case "anaerobic":
		return -1.0, -0.2, RuleAnaerobicReduce
	case "wet-hulled":
		return -2.0, 0.0, RuleWetHulledCooler
	default:
		return 0.0, 0.0, ""
	}
}

func roastEspressoAdj(roast string) (deltaTemp, deltaRatio, deltaTime float64, rule RuleID) {
	switch roast {
	case "light":
		return 1.0, 0.2, 4.0, RuleLightRoastHighTemp
	case "dark":
		return -2.0, -0.2, -4.0, RuleDarkRoastLowTemp
	default:
		return 0.0, 0.0, 0.0, ""
	}
}

func roastPouroverAdj(roast string) pouroverRoastAdj {
	switch roast {
	case "light":
		return pouroverRoastAdj{Rule: RuleLightRoastHighTemp, DeltaTemp: 1.0, RatioVal: 16.0, RatioLo: 15.5, RatioHi: 17.0, DeltaTime: 15.0}
	case "dark":
		return pouroverRoastAdj{Rule: RuleDarkRoastLowTemp, DeltaTemp: -1.0, RatioVal: 15.0, RatioLo: 14.0, RatioHi: 15.5, DeltaTime: -15.0}
	default:
		return pouroverRoastAdj{RatioVal: 15.5, RatioLo: 15.0, RatioHi: 16.5}
	}
}

func processPouroverAdj(process string) (deltaRatio float64, rule RuleID) {
	switch process {
	case "natural":
		return -0.5, RuleNaturalLowerExtract
	case "honey":
		return -0.3, RuleHoneyAdjust
	case "anaerobic":
		return -0.5, RuleAnaerobicReduce
	default:
		return 0.0, ""
	}
}

func drinkRatioDelta(drink string) (float64, RuleID) {
	switch drink {
	case "americano", "cortado":
		return 0.1, RuleDrinkRatioAdjust
	case "flat white", "cappuccino":
		return 0.2, RuleDrinkRatioAdjust
	case "latte":
		return 0.3, RuleDrinkRatioAdjust
	default:
		return 0.0, ""
	}
}

func espressoPreinfusion(roast string) models.ParameterValue {
	switch roast {
	case "light":
		return pv(7.0, 5.0, 9.0)
	case "dark":
		return pv(4.0, 3.0, 6.0)
	default:
		return pv(6.0, 4.0, 8.0)
	}
}

func collectRules(rules ...RuleID) []RuleID {
	var out []RuleID
	for _, r := range rules {
		if r != "" {
			out = append(out, r)
		}
	}
	return out
}

func pv(value, lo, hi float64) models.ParameterValue {
	return models.ParameterValue{
		Value: round1(value),
		Range: [2]float64{round1(lo), round1(hi)},
	}
}

// round1 rounds to one decimal place. IEEE 754 binary math turns clean inputs
// like 18.5 * 2.4 into 44.400000000000006; round1 keeps the JSON response and
// span attributes clean without relying on every consumer to format correctly.
func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
