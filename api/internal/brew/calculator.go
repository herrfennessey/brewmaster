package brew

import (
	"fmt"
	"math"
	"strings"
	"time"

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

// pouroverRoastAdj bundles the pourover roast non-temp adjustments. Temperature
// for pourover is set by roastBasePourover; this struct carries ratio band and
// time delta only.
type pouroverRoastAdj struct {
	RatioVal, RatioLo, RatioHi, DeltaTime float64
}

// roastDateAdjustments bundles freshness-driven nudges to keep function
// signatures within golangci's nargs limit.
type roastDateAdjustments struct {
	Rule                                 RuleID
	DeltaTemp, DeltaTime, DeltaPreinfuse float64
}

// Temperature operating envelope. Below 86°C, espresso machines struggle to
// extract; above 96°C, scalding flavors and pre-extraction occur.
const (
	tempMinC = 86.0
	tempMaxC = 96.0
)

// ComputeParams deterministically calculates brew parameters for the given bean,
// method, drink, and reference time. The reference time is used to derive days
// since roast for the freshness nudge — pass time.Now() in production and a
// fixed time in tests.
func ComputeParams(bean *CanonicalBean, method, drink string, now time.Time) ComputedParams {
	drink = NormalizeDrink(drink)
	if method == "pourover" {
		return computePourover(bean, now)
	}
	return computeEspresso(bean, drink, now)
}

// FallbackReasoning generates a reasoning string from applied parameter rules
// for use when the LLM annotation call fails.
func FallbackReasoning(rules []RuleID) string {
	//nolint:gosec // G101 false positive — these are descriptive phrases, not credentials.
	phrases := map[RuleID]string{
		RuleHighAltitudeTemp:               "high altitude suggests dense beans requiring slightly higher brew temperature",
		RuleLightRoastHighTemp:             "light roast benefits from higher temperature and extended extraction",
		RuleDarkRoastLowTemp:               "dark roast uses lower temperature to avoid bitterness",
		RuleNaturalLowerExtract:            "natural process gets a tighter ratio to balance its inherent sweetness",
		RuleAnaerobicReduce:                "anaerobic process needs a slightly cooler, tighter shot to preserve fermentation character",
		RuleHoneyAdjust:                    "honey process calls for a slightly tighter ratio",
		RuleWetHulledCooler:                "wet-hulled process is brewed cooler for body clarity",
		RuleDrinkRatioAdjust:               "drink style requires a higher yield ratio",
		RuleVarietalSoft:                   "softer aromatic varietal is brewed slightly cooler to preserve top notes",
		RuleVarietalHard:                   "denser varietal needs a slightly higher temperature to extract fully",
		RuleRoastFresh:                     "very fresh beans are still degassing, so preinfusion is extended",
		RuleRoastStale:                     "older beans get a small temperature and time bump to extract what aromatics remain",
		RuleRoastPostPeak:                  "beans are past peak freshness; flavors may be flattening",
		RuleUnknownRoastMediumLightDefault: "no roast level provided; assuming medium-light, the population mean for unlabelled specialty coffee",
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

// computeEspresso assembles espresso parameters from the roast-primary base
// plus altitude/varietal/process/freshness deltas, clamped to the operating
// envelope.
func computeEspresso(bean *CanonicalBean, drink string, now time.Time) ComputedParams {
	const baseDose = 18.5
	const baseRatio = 2.2
	const baseTime = 30.0

	baseTemp, tempLo, tempHi, roastRule := roastBaseEspresso(bean.RoastLevel)
	altDelta, altRule := altitudeAdj(bean)
	varDelta, varRule := varietalAdj(bean.Varietal)
	procTempDelta, procRatioDelta, procRule := processEspressoAdj(bean.Process)
	roastRatio, roastTime := roastEspressoNonTempAdj(bean.RoastLevel)
	drinkRatio, drinkRule := drinkRatioDelta(drink)

	rd := roastDateAdj(DaysSinceRoast(bean, now))

	tempAdj := altDelta + varDelta + procTempDelta + rd.DeltaTemp
	temp := clampTemp(baseTemp + tempAdj)
	tLo := clampTemp(tempLo + tempAdj)
	tHi := clampTemp(tempHi + tempAdj)

	ratio := baseRatio + procRatioDelta + roastRatio + drinkRatio
	timeVal := baseTime + roastTime + rd.DeltaTime
	yieldVal := baseDose * ratio

	preinf := shiftPV(espressoPreinfusion(bean.RoastLevel), rd.DeltaPreinfuse)

	return ComputedParams{
		Version:      RulesetVersion,
		AppliedRules: collectRules(roastRule, altRule, varRule, procRule, drinkRule, rd.Rule),
		Params: models.BrewParams{
			DoseG:        pv(baseDose, 18.0, 20.0),
			YieldG:       pv(yieldVal, yieldVal-3, yieldVal+3),
			Ratio:        fmt.Sprintf("1:%.1f", ratio),
			TempC:        pv(temp, tLo, tHi),
			TimeS:        pv(timeVal, timeVal-4, timeVal+4),
			PreinfusionS: preinf,
		},
	}
}

// computePourover assembles pourover parameters with the same axes as espresso
// but a higher roast-base envelope (+1°C), longer contact times, and a wider
// dilution ratio band that varies with roast level.
func computePourover(bean *CanonicalBean, now time.Time) ComputedParams {
	const baseDose = 15.0
	const baseTime = 210.0
	const bloomS = 35.0

	baseTemp, tempLo, tempHi, roastRule := roastBasePourover(bean.RoastLevel)
	altDelta, altRule := altitudeAdj(bean)
	varDelta, varRule := varietalAdj(bean.Varietal)
	procTempDelta, procRatioDelta, procRule := processPouroverAdj(bean.Process)
	ra := roastPouroverNonTempAdj(bean.RoastLevel)

	rd := roastDateAdj(DaysSinceRoast(bean, now))

	tempAdj := altDelta + varDelta + procTempDelta + rd.DeltaTemp
	temp := clampTemp(baseTemp + tempAdj)
	tLo := clampTemp(tempLo + tempAdj)
	tHi := clampTemp(tempHi + tempAdj)

	ratio := ra.RatioVal + procRatioDelta
	ratioLo := ra.RatioLo + procRatioDelta
	ratioHi := ra.RatioHi + procRatioDelta
	timeVal := baseTime + ra.DeltaTime + rd.DeltaTime
	waterVal := baseDose * ratio

	preinf := pv(bloomS+rd.DeltaPreinfuse, 30.0+rd.DeltaPreinfuse, 45.0+rd.DeltaPreinfuse)

	return ComputedParams{
		Version:      RulesetVersion,
		AppliedRules: collectRules(roastRule, altRule, varRule, procRule, rd.Rule),
		Params: models.BrewParams{
			DoseG:        pv(baseDose, 14.0, 17.0),
			YieldG:       pv(waterVal, baseDose*ratioLo, baseDose*ratioHi),
			Ratio:        fmt.Sprintf("1:%.1f", ratio),
			TempC:        pv(temp, tLo, tHi),
			TimeS:        pv(timeVal, timeVal-30, timeVal+30),
			PreinfusionS: preinf,
		},
	}
}

// roastBaseEspresso is the primary axis: roast level dictates extraction
// temperature, with light/medium/dark spread by ~4°C and an explicit
// medium-light band for beans roasted between light and full medium. Unknown
// roast level defaults to medium-light because most specialty roasters do not
// publish a roast level and the population mean of unlabelled lots skews
// medium-light.
func roastBaseEspresso(roast string) (base, lo, hi float64, rule RuleID) {
	switch roast {
	case "light":
		return 94.0, 93.0, 95.5, RuleLightRoastHighTemp
	case "medium-light":
		return 93.0, 92.0, 94.0, ""
	case "medium":
		return 92.0, 91.0, 93.0, ""
	case "dark":
		return 90.0, 88.5, 91.5, RuleDarkRoastLowTemp
	default: // unknown — apply medium-light default and surface the assumption
		return 93.0, 92.0, 94.0, RuleUnknownRoastMediumLightDefault
	}
}

// roastBasePourover mirrors roastBaseEspresso shifted +1°C across the board —
// pourover lacks pressure-driven extraction, so contact time and temperature do
// the work.
func roastBasePourover(roast string) (base, lo, hi float64, rule RuleID) {
	switch roast {
	case "light":
		return 95.0, 94.0, 96.0, RuleLightRoastHighTemp
	case "medium-light":
		return 94.0, 93.0, 95.0, ""
	case "medium":
		return 93.0, 92.0, 94.0, ""
	case "dark":
		return 91.0, 89.5, 92.5, RuleDarkRoastLowTemp
	default:
		return 94.0, 93.0, 95.0, RuleUnknownRoastMediumLightDefault
	}
}

// altitudeAdj is the density-by-altitude axis. Bounded at ±1°C; high-altitude
// bands (≥1800m) attach a rule for telemetry, others stay quiet.
func altitudeAdj(bean *CanonicalBean) (float64, RuleID) {
	if !bean.AltitudeKnown {
		return 0.0, ""
	}
	switch {
	case bean.AltitudeM < 1000:
		return -0.5, ""
	case bean.AltitudeM < 1400:
		return 0.0, ""
	case bean.AltitudeM < 1800:
		return 0.5, ""
	default:
		return 1.0, RuleHighAltitudeTemp
	}
}

// softVarietals are aromatic, lower-density cultivars that benefit from a
// slightly cooler brew to preserve volatiles. Substring match handles
// multi-varietal strings like "bourbon and typica".
var softVarietals = []string{
	"bourbon", "gesha", "sl28", "sl34",
	"pacamara", "maragogype", "eugenioides",
}

// hardVarietals are physically dense / different-species cultivars (mainly
// robusta and liberica) that need extra thermal energy to extract.
var hardVarietals = []string{"robusta", "liberica"}

// varietalAdj is the density-by-varietal axis. Substring match is used so
// composite strings ("bourbon, typica") still hit the appropriate bucket;
// unmatched varietals are treated as baseline (typica/caturra/catuai/etc.).
func varietalAdj(varietal string) (float64, RuleID) {
	if varietal == "" {
		return 0.0, ""
	}
	for _, v := range softVarietals {
		if strings.Contains(varietal, v) {
			return -0.5, RuleVarietalSoft
		}
	}
	for _, v := range hardVarietals {
		if strings.Contains(varietal, v) {
			return 1.0, RuleVarietalHard
		}
	}
	return 0.0, ""
}

// processEspressoAdj returns the temp + ratio deltas for a given process. Temp
// deltas are deliberately small — modern barista practice adjusts grind/ratio
// for natural and honey, and reserves temp adjustments for anaerobic and
// wet-hulled where flavor compounds are most heat-sensitive.
func processEspressoAdj(process string) (deltaTemp, deltaRatio float64, rule RuleID) {
	switch process {
	case "natural":
		return 0.0, -0.2, RuleNaturalLowerExtract
	case "honey":
		return 0.0, -0.1, RuleHoneyAdjust
	case "anaerobic":
		return -0.5, -0.2, RuleAnaerobicReduce
	case "wet-hulled":
		return -1.0, 0.0, RuleWetHulledCooler
	default:
		return 0.0, 0.0, ""
	}
}

// processPouroverAdj mirrors processEspressoAdj but with larger ratio swings,
// since pourover ratios live on a 1:14–1:17 scale where 0.3–0.5 shifts are
// equivalent to the ~0.1–0.2 shifts on espresso's 1:2 scale.
func processPouroverAdj(process string) (deltaTemp, deltaRatio float64, rule RuleID) {
	switch process {
	case "natural":
		return 0.0, -0.5, RuleNaturalLowerExtract
	case "honey":
		return 0.0, -0.3, RuleHoneyAdjust
	case "anaerobic":
		return -0.5, -0.5, RuleAnaerobicReduce
	case "wet-hulled":
		return -1.0, 0.0, RuleWetHulledCooler
	default:
		return 0.0, 0.0, ""
	}
}

// roastEspressoNonTempAdj returns the ratio and time deltas associated with a
// roast level. The temperature contribution lives in roastBaseEspresso; this
// function is non-temp only.
func roastEspressoNonTempAdj(roast string) (deltaRatio, deltaTime float64) {
	switch roast {
	case "light":
		return 0.2, 4.0
	case "medium-light":
		return 0.1, 2.0
	case "dark":
		return -0.2, -4.0
	default: // medium or unknown (treated as medium-light by default in temp axis)
		return 0.0, 0.0
	}
}

// roastPouroverNonTempAdj returns the absolute ratio band (not a delta — pourover
// ratios are roast-determined, not roast-modifier) and the time delta for a
// given roast level.
func roastPouroverNonTempAdj(roast string) pouroverRoastAdj {
	switch roast {
	case "light":
		return pouroverRoastAdj{RatioVal: 16.0, RatioLo: 15.5, RatioHi: 17.0, DeltaTime: 15.0}
	case "medium-light":
		return pouroverRoastAdj{RatioVal: 15.7, RatioLo: 15.2, RatioHi: 16.7, DeltaTime: 8.0}
	case "dark":
		return pouroverRoastAdj{RatioVal: 15.0, RatioLo: 14.0, RatioHi: 15.5, DeltaTime: -15.0}
	default:
		return pouroverRoastAdj{RatioVal: 15.5, RatioLo: 15.0, RatioHi: 16.5}
	}
}

// drinkRatioDelta opens up the espresso ratio for milk-forward drinks. This
// follows the "long shot for milk" school (Square Mile / Workshop / UK
// third-wave). The opposing "short shot for milk" school (Italian / Hedrick)
// pulls tighter ratios to punch through milk; both are defensible — we pick one
// and stay consistent.
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

// roastDateAdj is the freshness axis. Very fresh (<5d) beans are still off-
// gassing CO₂ and benefit from extended preinfusion to wet up evenly without
// channeling. Stale (>60d) beans have lost volatiles, so a small temperature
// and time bump pulls more out of what remains. The 35–60d "post-peak" band
// surfaces a flag without changing numbers.
func roastDateAdj(daysSinceRoast int, known bool) roastDateAdjustments {
	if !known {
		return roastDateAdjustments{}
	}
	switch {
	case daysSinceRoast < 5:
		return roastDateAdjustments{Rule: RuleRoastFresh, DeltaTime: 1.0, DeltaPreinfuse: 2.0}
	case daysSinceRoast > 60:
		return roastDateAdjustments{Rule: RuleRoastStale, DeltaTemp: 1.0, DeltaTime: 2.0}
	case daysSinceRoast > 35:
		return roastDateAdjustments{Rule: RuleRoastPostPeak}
	default:
		return roastDateAdjustments{}
	}
}

func espressoPreinfusion(roast string) models.ParameterValue {
	switch roast {
	case "light":
		return pv(7.0, 5.0, 9.0)
	case "medium-light":
		return pv(6.5, 4.5, 8.5)
	case "dark":
		return pv(4.0, 3.0, 6.0)
	default: // medium or unknown
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

// shiftPV shifts a ParameterValue by a delta on both the value and range,
// preserving the band width. Used for freshness nudges to preinfusion.
func shiftPV(p models.ParameterValue, delta float64) models.ParameterValue {
	return models.ParameterValue{
		Value: round1(p.Value + delta),
		Range: [2]float64{round1(p.Range[0] + delta), round1(p.Range[1] + delta)},
	}
}

// clampTemp keeps the temperature within the operating envelope of common
// espresso/pourover equipment so additive deltas can't compound out of range.
func clampTemp(t float64) float64 {
	if t < tempMinC {
		return round1(tempMinC)
	}
	if t > tempMaxC {
		return round1(tempMaxC)
	}
	return round1(t)
}

// round1 rounds to one decimal place. IEEE 754 binary math turns clean inputs
// like 18.5 * 2.4 into 44.400000000000006; round1 keeps the JSON response and
// span attributes clean without relying on every consumer to format correctly.
func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
