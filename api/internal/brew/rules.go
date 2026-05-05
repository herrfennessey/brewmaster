// Package brew provides deterministic brew parameter computation.
package brew

// RulesetVersion identifies the current parameter ruleset for traceability.
//
// v2 (current): roast-primary temperature model (light/medium-light/medium/dark
// bases with altitude + varietal + process as small additive deltas, clamped
// to [86, 96]°C). Adds varietal density axis, roast-date nudge, and a
// medium-light default for unlabeled beans.
//
// v1: altitude-primary temperature model with large process/roast deltas; could
// compound below operating range for dark+wet-hulled+low-altitude inputs.
const RulesetVersion = "v2"

// RuleID identifies a specific rule applied during computation.
type RuleID string

// Suitability poor rules — non-negotiable overrides that fire first.
//
//nolint:gosec
const (
	RuleGeshaMilk        RuleID = "v2.gesha_any_milk_poor"
	RuleEugenioidsMilk   RuleID = "v2.eugenioides_medium_high_milk_poor"
	RuleAnaerobicLatte   RuleID = "v2.anaerobic_latte_poor"
	RuleAnaerobicMedMilk RuleID = "v2.anaerobic_medium_milk_poor"
	RuleLightLatte       RuleID = "v2.light_latte_poor"
	RuleDarkPourover     RuleID = "v2.dark_pourover_poor"
)

// Suitability suboptimal rules — apply when no poor rule matched.
const (
	RuleLightMedMilk         RuleID = "v2.light_medium_milk_suboptimal"
	RuleEastAfricanLatte     RuleID = "v2.east_african_latte_suboptimal"
	RuleSL28MedHighMilk      RuleID = "v2.sl28_medium_high_milk_suboptimal"
	RuleNaturalAnaeroCortado RuleID = "v2.anaerobic_low_milk_suboptimal"
	RuleDarkCafeAuLait       RuleID = "v2.dark_cafe_au_lait_suboptimal"
)

// Suitability ideal rules — genuinely synergistic, not just absence of problems.
const (
	RuleMilkOriginDarkRoast    RuleID = "v2.milk_origin_dark_roast_ideal"
	RuleHoneyMediumMilk        RuleID = "v2.honey_medium_roast_milk_ideal"
	RuleWetHulledMilk          RuleID = "v2.wet_hulled_milk_ideal"
	RuleWashedLightBlack       RuleID = "v2.washed_light_pourover_black_ideal"
	RuleWashedEastAfricanBlack RuleID = "v2.washed_east_african_espresso_ideal"
)

// Parameter adjustment rules recorded in AppliedRules for traceability.
//
//nolint:gosec
const (
	RuleHighAltitudeTemp    RuleID = "v2.high_altitude_temp"
	RuleLightRoastHighTemp  RuleID = "v2.light_roast_high_temp"
	RuleDarkRoastLowTemp    RuleID = "v2.dark_roast_low_temp"
	RuleNaturalLowerExtract RuleID = "v2.natural_lower_extraction"
	RuleAnaerobicReduce     RuleID = "v2.anaerobic_reduced_extraction"
	RuleHoneyAdjust         RuleID = "v2.honey_adjusted"
	RuleWetHulledCooler     RuleID = "v2.wet_hulled_cooler"
	RuleDrinkRatioAdjust    RuleID = "v2.drink_ratio_adjusted"

	// Density axis — bean varietal influences thermal mass / extraction kinetics.
	RuleVarietalSoft RuleID = "v2.varietal_soft_lower_temp"
	RuleVarietalHard RuleID = "v2.varietal_hard_higher_temp"

	// Roast-date nudge — freshness affects degassing and remaining solubles.
	RuleRoastFresh    RuleID = "v2.roast_very_fresh"
	RuleRoastStale    RuleID = "v2.roast_stale_boost"
	RuleRoastPostPeak RuleID = "v2.roast_post_peak"

	// Surfaces general degassing guidance when the user could not provide a
	// roast date (most bag photos hide it; some bags only print expiration).
	// Informational only — does not change parameters.
	RuleRoastDateUnknown RuleID = "v2.roast_date_unknown"

	// Surfaces the medium-light default applied when roast level is missing,
	// since most specialty roasters do not publish a roast level.
	RuleUnknownRoastMediumLightDefault RuleID = "v2.unknown_roast_default_medium_light"
)
