// Package brew provides deterministic brew parameter computation.
package brew

// RulesetVersion identifies the current parameter ruleset for traceability.
// Pinned at v1 until GA; the app has no users yet so there's nothing to
// version against. Bump only when the user explicitly asks.
const RulesetVersion = "v1"

// RuleID identifies a specific rule applied during computation.
type RuleID string

// Suitability poor rules — non-negotiable overrides that fire first.
//
//nolint:gosec
const (
	RuleGeshaMilk        RuleID = "v1.gesha_any_milk_poor"
	RuleEugenioidsMilk   RuleID = "v1.eugenioides_medium_high_milk_poor"
	RuleAnaerobicLatte   RuleID = "v1.anaerobic_latte_poor"
	RuleAnaerobicMedMilk RuleID = "v1.anaerobic_medium_milk_poor"
	RuleLightLatte       RuleID = "v1.light_latte_poor"
	RuleDarkPourover     RuleID = "v1.dark_pourover_poor"
	RuleFilterMilk       RuleID = "v1.filter_intended_milk_poor"
)

// Suitability suboptimal rules — apply when no poor rule matched.
const (
	RuleLightMedMilk         RuleID = "v1.light_medium_milk_suboptimal"
	RuleEastAfricanLatte     RuleID = "v1.east_african_latte_suboptimal"
	RuleSL28MedHighMilk      RuleID = "v1.sl28_medium_high_milk_suboptimal"
	RuleNaturalAnaeroCortado RuleID = "v1.anaerobic_low_milk_suboptimal"
	RuleDarkCafeAuLait       RuleID = "v1.dark_cafe_au_lait_suboptimal"
	RuleFilterEspresso       RuleID = "v1.filter_intended_espresso_suboptimal"
	RuleEspressoFilter       RuleID = "v1.espresso_intended_pourover_suboptimal"
)

// Suitability ideal rules — genuinely synergistic, not just absence of problems.
const (
	RuleMilkOriginDarkRoast    RuleID = "v1.milk_origin_dark_roast_ideal"
	RuleHoneyMediumMilk        RuleID = "v1.honey_medium_roast_milk_ideal"
	RuleWetHulledMilk          RuleID = "v1.wet_hulled_milk_ideal"
	RuleWashedLightBlack       RuleID = "v1.washed_light_pourover_black_ideal"
	RuleWashedEastAfricanBlack RuleID = "v1.washed_east_african_espresso_ideal"
)

// Parameter adjustment rules recorded in AppliedRules for traceability.
//
//nolint:gosec
const (
	RuleHighAltitudeTemp    RuleID = "v1.high_altitude_temp"
	RuleLightRoastHighTemp  RuleID = "v1.light_roast_high_temp"
	RuleDarkRoastLowTemp    RuleID = "v1.dark_roast_low_temp"
	RuleNaturalLowerExtract RuleID = "v1.natural_lower_extraction"
	RuleAnaerobicReduce     RuleID = "v1.anaerobic_reduced_extraction"
	RuleHoneyAdjust         RuleID = "v1.honey_adjusted"
	RuleWetHulledCooler     RuleID = "v1.wet_hulled_cooler"
	RuleDrinkRatioAdjust    RuleID = "v1.drink_ratio_adjusted"

	// Density axis — bean varietal influences thermal mass / extraction kinetics.
	RuleVarietalSoft RuleID = "v1.varietal_soft_lower_temp"
	RuleVarietalHard RuleID = "v1.varietal_hard_higher_temp"

	// Roast-date nudge — freshness affects degassing and remaining solubles.
	RuleRoastFresh    RuleID = "v1.roast_very_fresh"
	RuleRoastStale    RuleID = "v1.roast_stale_boost"
	RuleRoastPostPeak RuleID = "v1.roast_post_peak"

	// Surfaces general degassing guidance when the user could not provide a
	// roast date (most bag photos hide it; some bags only print expiration).
	// Informational only — does not change parameters.
	RuleRoastDateUnknown RuleID = "v1.roast_date_unknown"

	// Surfaces the medium-light default applied when roast level is missing,
	// since most specialty roasters do not publish a roast level.
	RuleUnknownRoastMediumLightDefault RuleID = "v1.unknown_roast_default_medium_light"
)
