package store

import (
	"testing"
	"time"

	"github.com/herrfennessey/brewmaster/api/internal/models"
)

func sptr(s string) *string       { return &s }
func iptr(i int) *int             { return &i }
func fptr(f float64) *float64     { return &f }
func tptr(t time.Time) *time.Time { return &t }

func TestMergeBeanProfile_DoesNotClobberRicherExisting(t *testing.T) {
	existing := models.BeanProfile{
		SourceType: "image+web",
		Parsed: models.ParsedBean{
			Producer:    sptr("Finca La Soledad"),
			RoasterName: sptr("Bonanza"),
			Process:     sptr("washed"),
			AltitudeM:   fptr(1800),
			FlavorNotes: []string{"jasmine", "peach", "honey"},
		},
	}
	incoming := models.BeanProfile{
		SourceType: "text",
		Parsed: models.ParsedBean{
			Producer:    sptr("La Soledad"), // shorter, less specific
			RoasterName: sptr("Bonanza"),
			Process:     sptr("washed"),
			// AltitudeM, FlavorNotes intentionally absent
		},
	}
	merged := mergeBeanProfile(&existing, &incoming)

	if *merged.Parsed.Producer != "Finca La Soledad" {
		t.Errorf("producer was clobbered: got %q", *merged.Parsed.Producer)
	}
	if merged.Parsed.AltitudeM == nil || *merged.Parsed.AltitudeM != 1800 {
		t.Error("altitude was clobbered or dropped")
	}
	if len(merged.Parsed.FlavorNotes) != 3 {
		t.Errorf("flavor_notes were clobbered: got %v", merged.Parsed.FlavorNotes)
	}
	if merged.SourceType != "image+web" {
		t.Errorf("source_type unexpectedly downgraded: got %q", merged.SourceType)
	}
}

func TestMergeBeanProfile_FillsMissingFromIncoming(t *testing.T) {
	existing := models.BeanProfile{
		SourceType: "text",
		Parsed: models.ParsedBean{
			RoasterName: sptr("Bonanza"),
			// Producer, Process, AltitudeM, FlavorNotes missing
		},
	}
	incoming := models.BeanProfile{
		SourceType: "image+web",
		Parsed: models.ParsedBean{
			RoasterName: sptr("Bonanza"),
			Producer:    sptr("Finca La Soledad"),
			Process:     sptr("washed"),
			AltitudeM:   fptr(1800),
			FlavorNotes: []string{"jasmine", "peach"},
		},
	}
	merged := mergeBeanProfile(&existing, &incoming)

	if *merged.Parsed.Producer != "Finca La Soledad" {
		t.Error("producer should have been filled from incoming")
	}
	if merged.Parsed.AltitudeM == nil || *merged.Parsed.AltitudeM != 1800 {
		t.Error("altitude should have been filled from incoming")
	}
	if len(merged.Parsed.FlavorNotes) != 2 {
		t.Error("flavor_notes should have been filled from incoming")
	}
	if merged.SourceType != "image+web" {
		t.Errorf("source_type should upgrade text → image+web, got %q", merged.SourceType)
	}
}

func TestMergeBeanProfile_DowngradeSourceTypeIsRefused(t *testing.T) {
	existing := models.BeanProfile{SourceType: "image+web"}
	incoming := models.BeanProfile{SourceType: "text"}
	merged := mergeBeanProfile(&existing, &incoming)
	if merged.SourceType != "image+web" {
		t.Errorf("expected source_type to stay at image+web, got %q", merged.SourceType)
	}
}

func TestMergeParsedBean_LotYearAndConfidence(t *testing.T) {
	existing := models.ParsedBean{LotYear: iptr(2025)}
	incoming := models.ParsedBean{
		LotYear:            iptr(2026),
		AltitudeConfidence: sptr("range"),
	}
	merged := mergeParsedBean(&existing, &incoming)
	if *merged.LotYear != 2025 {
		t.Errorf("expected existing lot year preserved, got %d", *merged.LotYear)
	}
	if merged.AltitudeConfidence == nil || *merged.AltitudeConfidence != "range" {
		t.Error("altitude_confidence should fill from incoming when existing is nil")
	}
}

func TestMergeParsedBean_IntendedUseFillsAndPreserves(t *testing.T) {
	// Existing profile lacks intent → fills from incoming.
	merged := mergeParsedBean(
		&models.ParsedBean{},
		&models.ParsedBean{IntendedUse: sptr("filter")},
	)
	if merged.IntendedUse == nil || *merged.IntendedUse != "filter" {
		t.Errorf("expected intended_use to fill from incoming, got %v", merged.IntendedUse)
	}

	// Existing profile has intent → preserved over a thinner re-parse.
	merged = mergeParsedBean(
		&models.ParsedBean{IntendedUse: sptr("filter")},
		&models.ParsedBean{},
	)
	if merged.IntendedUse == nil || *merged.IntendedUse != "filter" {
		t.Errorf("expected intended_use preserved, got %v", merged.IntendedUse)
	}
}

func TestNewestOpenBag_NilWhenNone(t *testing.T) {
	t1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)
	if got := newestOpenBag(nil); got != nil {
		t.Errorf("nil bags: expected nil summary, got %+v", got)
	}
	bags := []Bag{
		{BagID: "a", OpenedAt: t1, FinishedAt: tptr(t2)},
	}
	if got := newestOpenBag(bags); got != nil {
		t.Errorf("all-finished: expected nil, got %+v", got)
	}
}

func TestNewestOpenBag_PicksMostRecentlyOpened(t *testing.T) {
	t1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	bags := []Bag{
		{BagID: "old-finished", OpenedAt: t1, FinishedAt: tptr(t2)},
		{BagID: "open-mid", OpenedAt: t2},
		{BagID: "open-new", OpenedAt: t3},
	}
	got := newestOpenBag(bags)
	if got == nil {
		t.Fatal("expected an open bag")
	}
	if got.BagID != "open-new" {
		t.Errorf("expected open-new, got %s", got.BagID)
	}
}

func TestApplyBagFinished_FlipsAndUnflips(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	c := Coffee{Bags: []Bag{
		{BagID: "a", OpenedAt: now.Add(-72 * time.Hour)},
	}}
	if err := applyBagFinished(&c, "a", true, now); err != nil {
		t.Fatalf("finish failed: %v", err)
	}
	if c.Bags[0].FinishedAt == nil || !c.Bags[0].FinishedAt.Equal(now) {
		t.Errorf("expected FinishedAt=%v, got %v", now, c.Bags[0].FinishedAt)
	}
	if err := applyBagFinished(&c, "a", false, now); err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	if c.Bags[0].FinishedAt != nil {
		t.Errorf("expected FinishedAt nil after reopen, got %v", c.Bags[0].FinishedAt)
	}
}

func TestApplyBagFinished_UnknownBag(t *testing.T) {
	c := Coffee{Bags: []Bag{{BagID: "a"}}}
	err := applyBagFinished(&c, "missing", true, time.Now())
	if err == nil || err.Error() != ErrBagNotFound.Error() {
		t.Errorf("expected ErrBagNotFound, got %v", err)
	}
}
