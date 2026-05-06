// Package store persists user-scoped coffee data in Firestore.
package store

import (
	"time"

	"github.com/herrfennessey/brewmaster/api/internal/models"
)

// Bag represents one physical purchase of a coffee. Re-orders of the same
// canonical bag append a new Bag entry to the parent Coffee.
type Bag struct {
	OpenedAt   time.Time  `firestore:"opened_at" json:"opened_at"`
	FinishedAt *time.Time `firestore:"finished_at,omitempty" json:"finished_at,omitempty"`
	RoastDate  *string    `firestore:"roast_date,omitempty" json:"roast_date,omitempty"`
	BagID      string     `firestore:"bag_id" json:"bag_id"`
	SourceType string     `firestore:"source_type" json:"source_type"`
}

// Coffee is a single canonical bag of coffee for one user. The Firestore
// document id is the canonical key; multiple physical purchases share one
// document via Bags.
type Coffee struct {
	FirstSeenAt   time.Time          `firestore:"first_seen_at" json:"first_seen_at"`
	LastSeenAt    time.Time          `firestore:"last_seen_at" json:"last_seen_at"`
	BestSessionID *string            `firestore:"best_session_id,omitempty" json:"best_session_id,omitempty"`
	Rating        *int               `firestore:"rating,omitempty" json:"rating,omitempty"`
	Notes         *string            `firestore:"notes,omitempty" json:"notes,omitempty"`
	CanonicalKey  string             `firestore:"canonical_key" json:"canonical_key"`
	BeanProfile   models.BeanProfile `firestore:"bean_profile" json:"bean_profile"`
	Bags          []Bag              `firestore:"bags" json:"bags"`
	SessionCount  int                `firestore:"session_count" json:"session_count"`
}

// BeanCard is the trimmed bean view shown on the My Coffees list. The full
// BeanProfile (which includes flavor_notes and other long fields) is loaded
// only when the user opens a specific coffee.
type BeanCard struct {
	RoasterName   string `json:"roaster_name"`
	Producer      string `json:"producer"`
	OriginCountry string `json:"origin_country"`
	OriginRegion  string `json:"origin_region"`
	Process       string `json:"process"`
	RoastLevel    string `json:"roast_level"`
	Varietal      string `json:"varietal"`
}

// CoffeeSummary is the trimmed shape returned by list endpoints. The
// canonical_key is the coffee's identity — clients use it to navigate to the
// detail page.
type CoffeeSummary struct {
	LastSeenAt   time.Time `json:"last_seen_at"`
	Rating       *int      `json:"rating,omitempty"`
	CanonicalKey string    `json:"canonical_key"`
	BeanCard     BeanCard  `json:"bean_card"`
	SessionCount int       `json:"session_count"`
}

// UpsertInput is the payload accepted by the upsert handler. RoastDate may be
// empty when the bag did not advertise one.
type UpsertInput struct {
	Rating      *int
	Notes       *string
	RoastDate   string
	BeanProfile models.BeanProfile
}

// PatchInput is the payload accepted by the patch handler.
type PatchInput struct {
	Rating *int
	Notes  *string
}
