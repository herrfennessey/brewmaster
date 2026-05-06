package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"

	"github.com/herrfennessey/brewmaster/api/internal/models"
)

// ErrNotFound is returned when a requested document does not exist.
var ErrNotFound = errors.New("store: not found")

// Repo is the storage interface used by handlers. The Firestore implementation
// is in this package; tests can substitute a fake.
type Repo interface {
	UpsertCoffee(ctx context.Context, uid, canonicalKey string, in *UpsertInput, now time.Time) (Coffee, bool, error)
	ListCoffees(ctx context.Context, uid string) ([]CoffeeSummary, error)
	GetCoffee(ctx context.Context, uid, coffeeID string) (Coffee, error)
	PatchCoffee(ctx context.Context, uid, coffeeID string, patch PatchInput, now time.Time) (Coffee, error)
	LookupByKey(ctx context.Context, uid, canonicalKey string) (CoffeeSummary, bool, error)
}

// FirestoreRepo persists coffees in Firestore. The collection layout is:
//
//	users/{uid}/coffees/{coffeeId}
//
// where coffeeId is the canonical key (deterministic) so dedupe is a Get,
// not a query.
type FirestoreRepo struct {
	client *firestore.Client
}

// NewFirestoreRepo wraps an initialized Firestore client.
func NewFirestoreRepo(client *firestore.Client) *FirestoreRepo {
	return &FirestoreRepo{client: client}
}

func (r *FirestoreRepo) coffees(uid string) *firestore.CollectionRef {
	return r.client.Collection("users").Doc(uid).Collection("coffees")
}

// UpsertCoffee creates a new coffee or appends a fresh bag to an existing one.
// Returns the resulting Coffee and a flag indicating whether it was newly
// created (true) or merged into an existing record (false).
func (r *FirestoreRepo) UpsertCoffee(
	ctx context.Context,
	uid, canonicalKey string,
	in *UpsertInput,
	now time.Time,
) (Coffee, bool, error) {
	docRef := r.coffees(uid).Doc(canonicalKey)

	// Try the create-first path: first save of a new bag is one round trip.
	fresh := newCoffee(canonicalKey, in, now)
	if _, err := docRef.Create(ctx, fresh); err == nil {
		return fresh, true, nil
	} else if status.Code(err) != codes.AlreadyExists {
		return Coffee{}, false, fmt.Errorf("create coffee: %w", err)
	}

	// Existing coffee — append the new bag inside a transaction so the read
	// and write of Bags don't race with another upsert.
	var result Coffee
	err := r.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err != nil {
			return fmt.Errorf("get coffee: %w", err)
		}
		var existing Coffee
		if err := snap.DataTo(&existing); err != nil {
			return fmt.Errorf("decode coffee: %w", err)
		}
		existing.Bags = append(existing.Bags, newBag(in, now))
		existing.LastSeenAt = now
		// Only fill in fields the existing profile is missing — never let a
		// thinner re-parse (e.g. text-only) clobber a richer earlier one (e.g.
		// image+web). The user can still correct fields explicitly via Patch.
		existing.BeanProfile = mergeBeanProfile(&existing.BeanProfile, &in.BeanProfile)
		applyPatch(&existing, PatchInput{Rating: in.Rating, Notes: in.Notes})
		result = existing
		return tx.Set(docRef, existing) //nolint:wrapcheck
	})
	if err != nil {
		return Coffee{}, false, fmt.Errorf("upsert coffee: %w", err)
	}
	return result, false, nil
}

// ListCoffees returns the user's saved coffees ordered by last_seen_at desc.
func (r *FirestoreRepo) ListCoffees(ctx context.Context, uid string) ([]CoffeeSummary, error) {
	iter := r.coffees(uid).OrderBy("last_seen_at", firestore.Desc).Documents(ctx)
	defer iter.Stop()

	out := []CoffeeSummary{}
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterate coffees: %w", err)
		}
		var c Coffee
		if err := snap.DataTo(&c); err != nil {
			return nil, fmt.Errorf("decode coffee: %w", err)
		}
		out = append(out, summary(snap.Ref.ID, &c))
	}
	return out, nil
}

// GetCoffee fetches one coffee by id.
func (r *FirestoreRepo) GetCoffee(ctx context.Context, uid, coffeeID string) (Coffee, error) {
	snap, err := r.coffees(uid).Doc(coffeeID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return Coffee{}, ErrNotFound
	}
	if err != nil {
		return Coffee{}, fmt.Errorf("get coffee: %w", err)
	}
	var c Coffee
	if err := snap.DataTo(&c); err != nil {
		return Coffee{}, fmt.Errorf("decode coffee: %w", err)
	}
	return c, nil
}

// PatchCoffee mutates rating and/or notes. Intentionally does not bump
// last_seen_at — that field tracks bag activity (saves, finishes, sessions),
// not metadata edits, so adjusting a rating shouldn't reshuffle My Coffees.
func (r *FirestoreRepo) PatchCoffee(
	ctx context.Context,
	uid, coffeeID string,
	patch PatchInput,
	_ time.Time,
) (Coffee, error) {
	docRef := r.coffees(uid).Doc(coffeeID)

	updates := []firestore.Update{}
	if patch.Rating != nil {
		updates = append(updates, firestore.Update{Path: "rating", Value: *patch.Rating})
	}
	if patch.Notes != nil {
		updates = append(updates, firestore.Update{Path: "notes", Value: *patch.Notes})
	}
	if len(updates) == 0 {
		return r.GetCoffee(ctx, uid, coffeeID)
	}

	if _, err := docRef.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return Coffee{}, ErrNotFound
		}
		return Coffee{}, fmt.Errorf("update coffee: %w", err)
	}
	return r.GetCoffee(ctx, uid, coffeeID)
}

// LookupByKey checks whether a user already has a coffee for the given key.
func (r *FirestoreRepo) LookupByKey(ctx context.Context, uid, canonicalKey string) (CoffeeSummary, bool, error) {
	snap, err := r.coffees(uid).Doc(canonicalKey).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return CoffeeSummary{}, false, nil
	}
	if err != nil {
		return CoffeeSummary{}, false, fmt.Errorf("lookup coffee: %w", err)
	}
	var c Coffee
	if err := snap.DataTo(&c); err != nil {
		return CoffeeSummary{}, false, fmt.Errorf("decode coffee: %w", err)
	}
	return summary(snap.Ref.ID, &c), true, nil
}

func newCoffee(key string, in *UpsertInput, now time.Time) Coffee {
	c := Coffee{
		CanonicalKey: key,
		BeanProfile:  in.BeanProfile,
		Bags:         []Bag{newBag(in, now)},
		FirstSeenAt:  now,
		LastSeenAt:   now,
		SessionCount: 0,
	}
	applyPatch(&c, PatchInput{Rating: in.Rating, Notes: in.Notes})
	return c
}

func newBag(in *UpsertInput, now time.Time) Bag {
	bag := Bag{
		BagID:      uuid.NewString(),
		OpenedAt:   now,
		SourceType: in.BeanProfile.SourceType,
	}
	if in.RoastDate != "" {
		rd := in.RoastDate
		bag.RoastDate = &rd
	}
	return bag
}

func applyPatch(c *Coffee, p PatchInput) {
	if p.Rating != nil {
		v := *p.Rating
		c.Rating = &v
	}
	if p.Notes != nil {
		v := *p.Notes
		c.Notes = &v
	}
}

// sourceTypeRank ranks parse sources from least to most informative. Used by
// mergeBeanProfile to upgrade — but never downgrade — the recorded source.
var sourceTypeRank = map[string]int{
	"":          0,
	"text":      1,
	"url":       2,
	"image":     3,
	"image+web": 4,
}

// mergeBeanProfile fills nil/empty fields on existing from incoming. It never
// overwrites data the existing profile already has — a thinner re-parse must
// not erase a richer earlier one. Source/ID/CreatedAt stay on the existing
// (we're keeping the first-parse's identity), except source_type which
// upgrades when the incoming parse is richer.
func mergeBeanProfile(existing, incoming *models.BeanProfile) models.BeanProfile {
	merged := *existing
	if sourceTypeRank[incoming.SourceType] > sourceTypeRank[existing.SourceType] {
		merged.SourceType = incoming.SourceType
	}
	if existing.CanonicalKey == nil && incoming.CanonicalKey != nil {
		merged.CanonicalKey = incoming.CanonicalKey
	}
	merged.Parsed = mergeParsedBean(&existing.Parsed, &incoming.Parsed)
	return merged
}

func mergeParsedBean(existing, incoming *models.ParsedBean) models.ParsedBean {
	merged := *existing
	merged.Producer = pickStr(existing.Producer, incoming.Producer)
	merged.OriginCountry = pickStr(existing.OriginCountry, incoming.OriginCountry)
	merged.OriginRegion = pickStr(existing.OriginRegion, incoming.OriginRegion)
	merged.AltitudeM = pickFloat(existing.AltitudeM, incoming.AltitudeM)
	merged.AltitudeConfidence = pickStr(existing.AltitudeConfidence, incoming.AltitudeConfidence)
	merged.Varietal = pickStr(existing.Varietal, incoming.Varietal)
	merged.Process = pickStr(existing.Process, incoming.Process)
	merged.RoastLevel = pickStr(existing.RoastLevel, incoming.RoastLevel)
	merged.RoastDate = pickStr(existing.RoastDate, incoming.RoastDate)
	merged.RoasterName = pickStr(existing.RoasterName, incoming.RoasterName)
	merged.LotYear = pickInt(existing.LotYear, incoming.LotYear)
	if len(existing.FlavorNotes) == 0 && len(incoming.FlavorNotes) > 0 {
		merged.FlavorNotes = incoming.FlavorNotes
	}
	return merged
}

func pickStr(a, b *string) *string {
	if a != nil && *a != "" {
		return a
	}
	return b
}

func pickFloat(a, b *float64) *float64 {
	if a != nil {
		return a
	}
	return b
}

func pickInt(a, b *int) *int {
	if a != nil {
		return a
	}
	return b
}

func summary(_ string, c *Coffee) CoffeeSummary {
	return CoffeeSummary{
		CanonicalKey: c.CanonicalKey,
		BeanCard:     newBeanCard(&c.BeanProfile),
		Rating:       c.Rating,
		LastSeenAt:   c.LastSeenAt,
		SessionCount: c.SessionCount,
	}
}

func newBeanCard(b *models.BeanProfile) BeanCard {
	return BeanCard{
		RoasterName:   strOrEmpty(b.Parsed.RoasterName),
		Producer:      strOrEmpty(b.Parsed.Producer),
		OriginCountry: strOrEmpty(b.Parsed.OriginCountry),
		OriginRegion:  strOrEmpty(b.Parsed.OriginRegion),
		Process:       strOrEmpty(b.Parsed.Process),
		RoastLevel:    strOrEmpty(b.Parsed.RoastLevel),
		Varietal:      strOrEmpty(b.Parsed.Varietal),
	}
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
