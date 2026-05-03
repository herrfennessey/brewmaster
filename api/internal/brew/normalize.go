package brew

import (
	"strings"

	"github.com/herrfennessey/brewmaster/api/internal/models"
)

// CanonicalBean is a normalized view of a ParsedBean used for rule evaluation.
type CanonicalBean struct {
	Process       string // guaranteed enum value or ""
	RoastLevel    string // "light" | "medium" | "dark" | ""
	OriginCountry string // lowercase
	OriginRegion  string // lowercase
	Varietal      string // canonical lowercase (geisha→gesha)
	FlavorNotes   []string
	AltitudeM     float64
	AltitudeKnown bool
}

var varietalAliases = map[string]string{
	"geisha": "gesha",
}

// Normalize converts a ParsedBean into a CanonicalBean with alias resolution and casing normalization.
func Normalize(bean *models.ParsedBean) CanonicalBean {
	cb := CanonicalBean{FlavorNotes: bean.FlavorNotes}
	if bean.AltitudeM != nil {
		cb.AltitudeM = *bean.AltitudeM
		cb.AltitudeKnown = true
	}
	if bean.OriginCountry != nil {
		cb.OriginCountry = strings.ToLower(*bean.OriginCountry)
	}
	if bean.OriginRegion != nil {
		cb.OriginRegion = strings.ToLower(*bean.OriginRegion)
	}
	if bean.Varietal != nil {
		cb.Varietal = normalizeVarietal(*bean.Varietal)
	}
	if bean.Process != nil {
		cb.Process = normalizeProcess(*bean.Process)
	}
	if bean.RoastLevel != nil {
		cb.RoastLevel = normalizeRoast(*bean.RoastLevel)
	}
	return cb
}

// NormalizeDrink lowercases and normalizes Unicode variants in drink strings.
func NormalizeDrink(drink string) string {
	drink = strings.ToLower(drink)
	drink = strings.ReplaceAll(drink, "é", "e")
	drink = strings.ReplaceAll(drink, "è", "e")
	drink = strings.ReplaceAll(drink, "à", "a")
	return drink
}

func normalizeVarietal(v string) string {
	v = strings.ToLower(v)
	if alias, ok := varietalAliases[v]; ok {
		return alias
	}
	return v
}

func normalizeProcess(p string) string {
	p = strings.ToLower(p)
	if p == "giling basah" {
		return "wet-hulled"
	}
	return p
}

func normalizeRoast(r string) string {
	r = strings.ToLower(r)
	switch {
	case strings.Contains(r, "light"):
		return "light"
	case strings.Contains(r, "dark"):
		return "dark"
	default:
		return "medium"
	}
}
