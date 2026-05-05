package brew

import (
	"strings"
	"time"

	"github.com/herrfennessey/brewmaster/api/internal/models"
)

// CanonicalBean is a normalized view of a ParsedBean used for rule evaluation.
//
// RoastLevel is intentionally empty when the source has no roast information,
// so the calculator can apply a medium-light default while suitability rules
// (which check explicit values) avoid firing for an unknown roast.
type CanonicalBean struct {
	RoastDate     *time.Time
	Process       string // guaranteed enum value or ""
	RoastLevel    string // "light" | "medium-light" | "medium" | "dark" | "" (unknown)
	OriginCountry string // lowercase
	OriginRegion  string // lowercase
	Varietal      string // canonical lowercase (geisha→gesha, sl-28→sl28, etc.)
	FlavorNotes   []string
	AltitudeM     float64
	AltitudeKnown bool
}

// varietalAliases canonicalizes spelling variants. Density bucketing happens
// downstream in calculator.varietalAdj.
var varietalAliases = map[string]string{
	"geisha": "gesha",
	"sl-28":  "sl28",
	"sl 28":  "sl28",
	"sl-34":  "sl34",
	"sl 34":  "sl34",
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
	if bean.RoastDate != nil {
		if t, err := time.Parse("2006-01-02", strings.TrimSpace(*bean.RoastDate)); err == nil {
			cb.RoastDate = &t
		}
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

// DaysSinceRoast returns whole days between the bean's roast date and now,
// clamped to >=0. The bool reports whether a roast date was known.
func DaysSinceRoast(bean *CanonicalBean, now time.Time) (int, bool) {
	if bean.RoastDate == nil {
		return 0, false
	}
	delta := now.Sub(*bean.RoastDate)
	days := int(delta.Hours() / 24)
	if days < 0 {
		days = 0
	}
	return days, true
}

func normalizeVarietal(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if alias, ok := varietalAliases[v]; ok {
		return alias
	}
	return v
}

func normalizeProcess(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	if p == "giling basah" {
		return "wet-hulled"
	}
	return p
}

// normalizeRoast preserves an empty input as "" (unknown). Source values that
// don't match any known band also return "" so callers can apply a defensible
// default rather than mis-bucketing into "medium".
func normalizeRoast(r string) string {
	r = strings.ToLower(strings.TrimSpace(r))
	if r == "" {
		return ""
	}
	switch {
	case strings.Contains(r, "medium-light"), strings.Contains(r, "medium light"):
		return "medium-light"
	case strings.Contains(r, "light"):
		return "light"
	case strings.Contains(r, "dark"):
		return "dark"
	case strings.Contains(r, "medium"):
		return "medium"
	default:
		return ""
	}
}
