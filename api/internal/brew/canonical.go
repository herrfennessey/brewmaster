package brew

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/herrfennessey/brewmaster/api/internal/models"
)

// HashKey returns a 16-hex-char SHA-256 prefix of the canonical key. We use
// this as the Firestore document id and URL slug instead of the full
// roaster|bean|process string — which can stretch into the hundreds of
// percent-encoded characters once the producer field carries a long farm
// description. 16 hex chars (64 bits) gives a birthday-collision bound around
// 2^32 distinct keys per user, which is laughably safe at personal scale.
func HashKey(canonicalKey string) string {
	sum := sha256.Sum256([]byte(canonicalKey))
	return hex.EncodeToString(sum[:8])
}

// roasterSuffixes are dropped from a roaster slug after slugification so that
// "Stone & Wood Coffee Roasters" and "Stone & Wood" produce the same key.
var roasterSuffixes = []string{
	"coffee-roasters",
	"coffee-roastery",
	"coffee-roasting",
	"roasters",
	"roastery",
	"roasting-co",
	"roasting",
	"coffee-co",
	"coffee",
	"co",
}

// CanonicalKey returns a stable identifier for a bag of coffee suitable for
// dedupe across re-orders. The key is `roaster|bean|process`, where each
// component is slugified (NFKD-folded, lowercased, non-alphanumerics collapsed
// to `-`). It deliberately ignores origin and varietal because those wobble
// across listings of the same bag.
//
// Returns ok=false when the inputs are too thin to dedupe — the caller should
// then mint a fresh UUID and flag the record as manually identified.
func CanonicalKey(p *models.ParsedBean) (key string, ok bool) {
	roaster := slugifyRoaster(deref(p.RoasterName))
	bean := beanSlug(p)
	if roaster == "" || bean == "" {
		return "", false
	}

	process := normalizeProcess(deref(p.Process))
	if process == "" {
		process = "unknown"
	}
	process = slugify(process)

	return roaster + "|" + bean + "|" + process, true
}

// beanSlug picks the most stable identifier available for the bean itself,
// falling back through producer → origin region+country.
func beanSlug(p *models.ParsedBean) string {
	if s := slugify(deref(p.Producer)); s != "" {
		return s
	}
	region := deref(p.OriginRegion)
	country := deref(p.OriginCountry)
	combined := strings.TrimSpace(region + " " + country)
	return slugify(combined)
}

// slugifyRoaster slugifies and then strips trailing "coffee/roasters/co"
// fragments so naming variants collapse onto the same key.
func slugifyRoaster(s string) string {
	slug := slugify(s)
	for {
		trimmed := slug
		for _, suffix := range roasterSuffixes {
			trimmed = strings.TrimSuffix(trimmed, "-"+suffix)
			trimmed = strings.TrimSuffix(trimmed, suffix)
		}
		if trimmed == slug {
			break
		}
		slug = strings.Trim(trimmed, "-")
	}
	return slug
}

// slugify NFKD-folds combining marks, lowercases, and replaces every run of
// non-alphanumerics with a single `-`.
func slugify(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	folded, _, err := transform.String(
		transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)), norm.NFC),
		s,
	)
	if err != nil {
		folded = s
	}

	var b strings.Builder
	b.Grow(len(folded))
	prevDash := true
	for _, r := range folded {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
			prevDash = false
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
