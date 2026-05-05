package brew

import (
	"strings"

	"github.com/herrfennessey/brewmaster/api/internal/models"
)

// SuitabilityResult holds the deterministic suitability assessment.
type SuitabilityResult struct {
	models.DrinkSuitability
	Rule RuleID
}

var suitabilityReasons = map[RuleID]string{
	// Poor
	RuleGeshaMilk:        "Gesha's jasmine and tea-like complexity is overwhelmed by milk regardless of volume.",
	RuleEugenioidsMilk:   "Eugenioides is too delicate and rare to be lost in milk.",
	RuleAnaerobicLatte:   "Anaerobic fermentation character is buried under milk in a latte.",
	RuleAnaerobicMedMilk: "Anaerobic fermentation aromatics are lost in medium-milk espresso drinks.",
	RuleLightLatte:       "Light roast delicacy disappears completely in high milk volume.",
	RuleDarkPourover:     "Dark roast is already very soluble; pourover's long contact time over-extracts it into bitterness with no pressure-driven cutoff.",
	// Suboptimal
	RuleLightMedMilk:         "Light roast character is significantly diminished with medium milk volume.",
	RuleEastAfricanLatte:     "Ethiopian and Kenyan brightness and acidity — the whole point — are masked by milk.",
	RuleSL28MedHighMilk:      "SL28's signature sharp acidity does not integrate with milk; can taste sour.",
	RuleNaturalAnaeroCortado: "Anaerobic fermentation character loses nuance even in low-milk drinks compared to straight espresso.",
	RuleDarkCafeAuLait:       "Dark roast in cafe au lait is drinkable but heavy and flat.",
	// Ideal
	RuleMilkOriginDarkRoast:    "Chocolate and caramel body of this origin cuts through milk cleanly for a classic milk espresso.",
	RuleHoneyMediumMilk:        "Honey process body and sweetness synergise with milk in an ideal combination.",
	RuleWetHulledMilk:          "Wet-hulled earthy chocolate and very low acidity are purpose-built for milk drinks.",
	RuleWashedLightBlack:       "Washed light roast clarity and origin complexity shine without interference in black coffee.",
	RuleWashedEastAfricanBlack: "Washed Ethiopian and Kenyan coffee shows what specialty coffee is — bright fruit and florals expressed fully.",
}

const defaultSuitabilityReason = "This bean and drink are a standard pairing with no notable trade-offs."

var eastAfricanOrigins = map[string]bool{
	"ethiopia": true, "kenya": true, "rwanda": true, "burundi": true,
}

var milkFriendlyOrigins = map[string]bool{
	"brazil": true, "colombia": true, "guatemala": true,
	"el salvador": true, "indonesia": true,
}

// ComputeSuitability returns the deterministic suitability level for a bean-drink pairing.
// Rules are checked in priority order: poor → suboptimal → ideal → suitable (default).
func ComputeSuitability(bean *CanonicalBean, drink string) SuitabilityResult {
	drink = NormalizeDrink(drink)
	if rule, ok := checkPoorRules(bean, drink); ok {
		return buildSuitabilityResult("poor", rule)
	}
	if rule, ok := checkSuboptimalRules(bean, drink); ok {
		return buildSuitabilityResult("suboptimal", rule)
	}
	if rule, ok := checkIdealRules(bean, drink); ok {
		return buildSuitabilityResult("ideal", rule)
	}
	return SuitabilityResult{
		DrinkSuitability: models.DrinkSuitability{Level: "suitable", Reason: defaultSuitabilityReason},
	}
}

func buildSuitabilityResult(level string, rule RuleID) SuitabilityResult {
	reason, ok := suitabilityReasons[rule]
	if !ok {
		reason = defaultSuitabilityReason
	}
	return SuitabilityResult{
		DrinkSuitability: models.DrinkSuitability{Level: level, Reason: reason},
		Rule:             rule,
	}
}

func checkPoorRules(bean *CanonicalBean, drink string) (RuleID, bool) {
	if bean.Varietal == "gesha" && isAnyMilk(drink) {
		return RuleGeshaMilk, true
	}
	if bean.Varietal == "eugenioides" && (isMediumMilk(drink) || isHighMilk(drink)) {
		return RuleEugenioidsMilk, true
	}
	if isAnaerobicLike(bean) && isHighMilk(drink) {
		return RuleAnaerobicLatte, true
	}
	if isAnaerobicLike(bean) && isMediumMilk(drink) {
		return RuleAnaerobicMedMilk, true
	}
	if bean.RoastLevel == "light" && isHighMilk(drink) {
		return RuleLightLatte, true
	}
	if bean.RoastLevel == "dark" && drink == "black" {
		return RuleDarkPourover, true
	}
	return "", false
}

func checkSuboptimalRules(bean *CanonicalBean, drink string) (RuleID, bool) {
	if bean.RoastLevel == "light" && isMediumMilk(drink) {
		return RuleLightMedMilk, true
	}
	// East African brightness only conflicts with milk while the roast is still
	// light enough to preserve it; dark-roasted Ethiopian Sidamo in a latte is a
	// classic, defensible pairing.
	if eastAfricanOrigins[bean.OriginCountry] && isLightishRoast(bean.RoastLevel) && isHighMilk(drink) {
		return RuleEastAfricanLatte, true
	}
	// SL28's sharp blackcurrant acidity comes from washed processing; pulped
	// natural / honey SL28 integrates with milk just fine.
	if bean.Varietal == "sl28" && bean.Process == "washed" && (isMediumMilk(drink) || isHighMilk(drink)) {
		return RuleSL28MedHighMilk, true
	}
	// Only anaerobic / loud-fermentation lots lose nuance in low-milk drinks;
	// regular naturals (Brazil, Ethiopia sun-dried) work fine in cortados.
	if isAnaerobicLike(bean) && isLowMilk(drink) {
		return RuleNaturalAnaeroCortado, true
	}
	if bean.RoastLevel == "dark" && drink == "cafe au lait" {
		return RuleDarkCafeAuLait, true
	}
	return "", false
}

// isLightishRoast covers explicit light, medium-light, and medium roasts —
// the ones where origin character is still front-and-center. Dark and unknown
// are excluded.
func isLightishRoast(roast string) bool {
	return roast == "light" || roast == "medium-light" || roast == "medium"
}

func checkIdealRules(bean *CanonicalBean, drink string) (RuleID, bool) {
	if rule, ok := checkIdealMilkRules(bean, drink); ok {
		return rule, ok
	}
	return checkIdealBlackRules(bean, drink)
}

func checkIdealMilkRules(bean *CanonicalBean, drink string) (RuleID, bool) {
	if milkFriendlyOrigins[bean.OriginCountry] && bean.RoastLevel == "dark" && isAnyMilk(drink) {
		return RuleMilkOriginDarkRoast, true
	}
	if bean.Process == "honey" && bean.RoastLevel == "medium" && isAnyMilk(drink) {
		return RuleHoneyMediumMilk, true
	}
	if bean.Process == "wet-hulled" && (isMediumMilk(drink) || isHighMilk(drink)) {
		return RuleWetHulledMilk, true
	}
	return "", false
}

func checkIdealBlackRules(bean *CanonicalBean, drink string) (RuleID, bool) {
	// East African washed espresso/americano checked before generic washed-light rule
	// to provide the more specific reason string.
	if bean.Process == "washed" && eastAfricanOrigins[bean.OriginCountry] && (drink == "espresso" || drink == "americano") {
		return RuleWashedEastAfricanBlack, true
	}
	isLightOrMed := bean.RoastLevel == "light" || bean.RoastLevel == "medium"
	if bean.Process == "washed" && isLightOrMed && (drink == "black" || drink == "espresso") {
		return RuleWashedLightBlack, true
	}
	return "", false
}

func isHighMilk(drink string) bool   { return drink == "latte" }
func isMediumMilk(drink string) bool { return drink == "cappuccino" || drink == "flat white" }
func isLowMilk(drink string) bool    { return drink == "cortado" || drink == "macchiato" }
func isAnyMilk(drink string) bool {
	return isHighMilk(drink) || isMediumMilk(drink) || isLowMilk(drink)
}

func isAnaerobicLike(bean *CanonicalBean) bool {
	if bean.Process == "anaerobic" {
		return true
	}
	if bean.Process != "other" {
		return false
	}
	fermentWords := []string{"ferment", "carbonic", "wine", "funky"}
	for _, note := range bean.FlavorNotes {
		noteLower := strings.ToLower(note)
		for _, word := range fermentWords {
			if strings.Contains(noteLower, word) {
				return true
			}
		}
	}
	return false
}
