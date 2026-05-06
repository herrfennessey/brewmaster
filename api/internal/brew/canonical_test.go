package brew_test

import (
	"testing"

	"github.com/herrfennessey/brewmaster/api/internal/brew"
	"github.com/herrfennessey/brewmaster/api/internal/models"
)

func TestCanonicalKey_BasicTriple(t *testing.T) {
	bean := models.ParsedBean{
		RoasterName: strPtr("Bonanza Coffee Roasters"),
		Producer:    strPtr("Finca La Soledad"),
		Process:     strPtr("Washed"),
	}
	got, ok := brew.CanonicalKey(&bean)
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := "bonanza|finca-la-soledad|washed"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCanonicalKey_StripsRoasterSuffix(t *testing.T) {
	tests := []struct {
		roaster string
		want    string
	}{
		{"Bonanza Coffee Roasters", "bonanza"},
		{"Bonanza Coffee", "bonanza"},
		{"Bonanza Roastery", "bonanza"},
		{"Bonanza Coffee Co.", "bonanza"},
		{"Bonanza Coffee Roasting Co.", "bonanza"},
		{"Bonanza", "bonanza"},
	}
	for _, tt := range tests {
		bean := models.ParsedBean{
			RoasterName: strPtr(tt.roaster),
			Producer:    strPtr("X"),
			Process:     strPtr("washed"),
		}
		got, ok := brew.CanonicalKey(&bean)
		if !ok {
			t.Errorf("input %q: expected ok=true", tt.roaster)
			continue
		}
		want := tt.want + "|x|washed"
		if got != want {
			t.Errorf("input %q: got %q, want %q", tt.roaster, got, want)
		}
	}
}

func TestCanonicalKey_AccentFolding(t *testing.T) {
	bean := models.ParsedBean{
		RoasterName: strPtr("Café Crème"),
		Producer:    strPtr("Hacienda La Esméralda"),
		Process:     strPtr("Natural"),
	}
	got, ok := brew.CanonicalKey(&bean)
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := "cafe-creme|hacienda-la-esmeralda|natural"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCanonicalKey_PunctuationCollapsesToDash(t *testing.T) {
	bean := models.ParsedBean{
		RoasterName: strPtr("Square Mile"),
		Producer:    strPtr("AA / Lot #42 — Top Grade"),
		Process:     strPtr("washed"),
	}
	got, ok := brew.CanonicalKey(&bean)
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := "square-mile|aa-lot-42-top-grade|washed"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCanonicalKey_GilingBasahNormalized(t *testing.T) {
	bean := models.ParsedBean{
		RoasterName: strPtr("Klatch"),
		Producer:    strPtr("Gayo"),
		Process:     strPtr("Giling Basah"),
	}
	got, ok := brew.CanonicalKey(&bean)
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := "klatch|gayo|wet-hulled"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCanonicalKey_MissingProcessBecomesUnknown(t *testing.T) {
	bean := models.ParsedBean{
		RoasterName: strPtr("Bonanza"),
		Producer:    strPtr("X"),
	}
	got, ok := brew.CanonicalKey(&bean)
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := "bonanza|x|unknown"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCanonicalKey_BeanFallsBackToOrigin(t *testing.T) {
	bean := models.ParsedBean{
		RoasterName:   strPtr("Bonanza"),
		OriginCountry: strPtr("Ethiopia"),
		OriginRegion:  strPtr("Yirgacheffe"),
		Process:       strPtr("Washed"),
	}
	got, ok := brew.CanonicalKey(&bean)
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := "bonanza|yirgacheffe-ethiopia|washed"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCanonicalKey_NoRoasterReturnsNotOK(t *testing.T) {
	bean := models.ParsedBean{
		Producer: strPtr("Finca X"),
		Process:  strPtr("Washed"),
	}
	if _, ok := brew.CanonicalKey(&bean); ok {
		t.Error("expected ok=false when roaster is missing")
	}
}

func TestCanonicalKey_NoBeanIdentifierReturnsNotOK(t *testing.T) {
	bean := models.ParsedBean{
		RoasterName: strPtr("Bonanza"),
		Process:     strPtr("Washed"),
	}
	if _, ok := brew.CanonicalKey(&bean); ok {
		t.Error("expected ok=false when no producer or origin is available")
	}
}

func TestCanonicalKey_EmptyStringsReturnsNotOK(t *testing.T) {
	empty := ""
	bean := models.ParsedBean{
		RoasterName: &empty,
		Producer:    &empty,
		Process:     &empty,
	}
	if _, ok := brew.CanonicalKey(&bean); ok {
		t.Error("expected ok=false when all strings are empty")
	}
}

func TestCanonicalKey_StableAcrossCasingAndWhitespace(t *testing.T) {
	a := models.ParsedBean{
		RoasterName: strPtr("  BONANZA Coffee Roasters  "),
		Producer:    strPtr("FINCA  La  Soledad"),
		Process:     strPtr("WASHED"),
	}
	b := models.ParsedBean{
		RoasterName: strPtr("bonanza coffee"),
		Producer:    strPtr("finca la soledad"),
		Process:     strPtr("washed"),
	}
	keyA, okA := brew.CanonicalKey(&a)
	keyB, okB := brew.CanonicalKey(&b)
	if !okA || !okB {
		t.Fatalf("expected both ok, got %v %v", okA, okB)
	}
	if keyA != keyB {
		t.Errorf("expected stable key, got %q vs %q", keyA, keyB)
	}
}

func TestCanonicalKey_RoasterEmptyAfterSuffixStripReturnsNotOK(t *testing.T) {
	bean := models.ParsedBean{
		RoasterName: strPtr("Coffee Roasters"),
		Producer:    strPtr("X"),
		Process:     strPtr("washed"),
	}
	if _, ok := brew.CanonicalKey(&bean); ok {
		t.Error("expected ok=false when roaster is just a suffix word")
	}
}

func TestHashKey_StableAndShort(t *testing.T) {
	cases := []string{
		"bonanza|finca-la-soledad|washed",
		"five-elephant|gakenke-washing-station-farm-smallhold-farmers-of-the-kayanza-region|washed",
		"",
	}
	seen := map[string]string{}
	for _, in := range cases {
		got := brew.HashKey(in)
		if len(got) != 16 {
			t.Errorf("HashKey(%q) = %q, want 16 hex chars", in, got)
		}
		// determinism
		if again := brew.HashKey(in); again != got {
			t.Errorf("HashKey(%q) not deterministic: %q vs %q", in, got, again)
		}
		seen[in] = got
	}
	// distinctness — different slugs must produce different ids in this small set
	if seen[cases[0]] == seen[cases[1]] {
		t.Error("expected distinct hashes for distinct inputs")
	}
}
