package brew

import (
	"fmt"
	"strings"

	"github.com/herrfennessey/brewmaster/api/internal/models"
)

// ComputeConfidence returns a weighted confidence level based on bean profile completeness.
//
// Field weights (total = 20):
//   - roast_level: 5, process: 4, altitude_m: 4, origin_country: 3, varietal: 2,
//     origin_region: 1, flavor_notes (non-empty): 1
//
// ≥16 → high, 10–15 → medium, <10 → low.
func ComputeConfidence(bean *CanonicalBean) models.BrewConfidence {
	score := 20
	var missing []string

	if bean.RoastLevel == "" {
		score -= 5
		missing = append(missing, "roast level")
	}
	if bean.Process == "" {
		score -= 4
		missing = append(missing, "process")
	}
	if !bean.AltitudeKnown {
		score -= 4
		missing = append(missing, "altitude")
	}
	if bean.OriginCountry == "" {
		score -= 3
		missing = append(missing, "origin country")
	}
	if bean.Varietal == "" {
		score -= 2
		missing = append(missing, "varietal")
	}
	if bean.OriginRegion == "" {
		score--
	}
	if len(bean.FlavorNotes) == 0 {
		score--
	}

	return models.BrewConfidence{
		Level:  confidenceLevel(score),
		Reason: confidenceReason(missing),
	}
}

func confidenceLevel(score int) string {
	switch {
	case score >= 16:
		return "high"
	case score >= 10:
		return "medium"
	default:
		return "low"
	}
}

func confidenceReason(missing []string) string {
	if len(missing) == 0 {
		return "Complete bean profile with all key fields present."
	}
	return fmt.Sprintf("Missing %s; parameters use conservative defaults.", strings.Join(missing, ", "))
}
