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

	"github.com/herrfennessey/brewmaster/api/internal/brew"
	"github.com/herrfennessey/brewmaster/api/internal/models"
)

// ErrNotFound is returned when a requested document does not exist.
var ErrNotFound = errors.New("store: not found")

// Repo is the storage interface used by handlers. The Firestore implementation
// is in this package; tests can substitute a fake. The store owns the mapping
// from canonical-key (long roaster|bean|process slug) to coffeeID (short hash
// from brew.HashKey), so handlers don't repeat the derivation.
type Repo interface {
	UpsertCoffee(ctx context.Context, uid, canonicalKey string, in *UpsertInput, now time.Time) (CoffeeSummary, Coffee, bool, error)
	ListCoffees(ctx context.Context, uid string) ([]CoffeeSummary, error)
	GetCoffee(ctx context.Context, uid, coffeeID string) (Coffee, error)
	PatchCoffee(ctx context.Context, uid, coffeeID string, patch PatchInput, now time.Time) (Coffee, error)
	DeleteCoffee(ctx context.Context, uid, coffeeID string) error
	SetBagFinished(ctx context.Context, uid, coffeeID, bagID string, finished bool, now time.Time) (Coffee, error)
	LookupBySlug(ctx context.Context, uid, canonicalKey string) (CoffeeSummary, bool, error)
}

// FirestoreRepo persists coffees in Firestore. The collection layout is:
//
//	users/{uid}/coffees/{coffeeId}
//
// where coffeeId is brew.HashKey(canonicalKey). The full canonicalKey lives
// inside the document so it survives in backups even if the hash strategy
// changes later.
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
// Returns a summary (with the derived coffee_id), the full coffee doc, and a
// flag indicating whether it was newly created (true) or merged (false).
func (r *FirestoreRepo) UpsertCoffee(
	ctx context.Context,
	uid, canonicalKey string,
	in *UpsertInput,
	now time.Time,
) (CoffeeSummary, Coffee, bool, error) {
	coffeeID := brew.HashKey(canonicalKey)
	docRef := r.coffees(uid).Doc(coffeeID)

	// Try the create-first path: first save of a new bag is one round trip.
	fresh := newCoffee(canonicalKey, in, now)
	if _, err := docRef.Create(ctx, fresh); err == nil {
		sum := summary(coffeeID, &fresh)
		return sum, fresh, true, nil
	} else if status.Code(err) != codes.AlreadyExists {
		return CoffeeSummary{}, Coffee{}, false, fmt.Errorf("create coffee: %w", err)
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
		return CoffeeSummary{}, Coffee{}, false, fmt.Errorf("upsert coffee: %w", err)
	}
	sum := summary(coffeeID, &result)
	return sum, result, false, nil
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

// DeleteCoffee removes one coffee document. Firestore's Delete is idempotent
// (no-op when the doc is already gone), so we Get first to surface ErrNotFound
// — callers want to distinguish "deleted now" from "wasn't there".
func (r *FirestoreRepo) DeleteCoffee(ctx context.Context, uid, coffeeID string) error {
	docRef := r.coffees(uid).Doc(coffeeID)
	if _, err := docRef.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return fmt.Errorf("get coffee: %w", err)
	}
	if _, err := docRef.Delete(ctx); err != nil {
		return fmt.Errorf("delete coffee: %w", err)
	}
	return nil
}

// GetCoffee fetches one coffee by its hash id.
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
	for _, field := range patch.Clear {
		updates = append(updates, firestore.Update{Path: field, Value: firestore.Delete})
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

// ErrBagNotFound is returned when SetBagFinished can't find the bag inside the
// coffee. Distinct from ErrNotFound (which means the coffee itself is missing)
// so handlers can render a more useful error.
var ErrBagNotFound = errors.New("store: bag not found")

// SetBagFinished flips a single bag's finished_at field on or off. Bags live
// inline on the Coffee doc, so this reads, mutates, and writes back inside a
// transaction to avoid racing a concurrent upsert (which appends bags).
func (r *FirestoreRepo) SetBagFinished(
	ctx context.Context,
	uid, coffeeID, bagID string,
	finished bool,
	now time.Time,
) (Coffee, error) {
	docRef := r.coffees(uid).Doc(coffeeID)
	var result Coffee
	err := r.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		c, err := readCoffeeForBagUpdate(tx, docRef)
		if err != nil {
			return err
		}
		if err := applyBagFinished(&c, bagID, finished, now); err != nil {
			return err
		}
		result = c
		return tx.Set(docRef, c) //nolint:wrapcheck
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Coffee{}, ErrNotFound
		}
		if errors.Is(err, ErrBagNotFound) {
			return Coffee{}, ErrBagNotFound
		}
		return Coffee{}, fmt.Errorf("set bag finished: %w", err)
	}
	return result, nil
}

func readCoffeeForBagUpdate(tx *firestore.Transaction, docRef *firestore.DocumentRef) (Coffee, error) {
	snap, err := tx.Get(docRef)
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

func applyBagFinished(c *Coffee, bagID string, finished bool, now time.Time) error {
	for i := range c.Bags {
		if c.Bags[i].BagID != bagID {
			continue
		}
		if finished {
			t := now
			c.Bags[i].FinishedAt = &t
		} else {
			c.Bags[i].FinishedAt = nil
		}
		return nil
	}
	return ErrBagNotFound
}

// LookupBySlug checks whether a user already has a coffee for the given
// canonical key (slug). The hash strategy stays encapsulated here.
func (r *FirestoreRepo) LookupBySlug(ctx context.Context, uid, canonicalKey string) (CoffeeSummary, bool, error) {
	snap, err := r.coffees(uid).Doc(brew.HashKey(canonicalKey)).Get(ctx)
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
	merged.IntendedUse = pickStr(existing.IntendedUse, incoming.IntendedUse)
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

func summary(coffeeID string, c *Coffee) CoffeeSummary {
	return CoffeeSummary{
		CoffeeID:     coffeeID,
		BeanCard:     newBeanCard(&c.BeanProfile),
		Rating:       c.Rating,
		LastSeenAt:   c.LastSeenAt,
		SessionCount: c.SessionCount,
		BagCount:     len(c.Bags),
		OpenBag:      newestOpenBag(c.Bags),
	}
}

// newestOpenBag picks the most-recently-opened bag where finished_at is nil.
// One coffee can carry many bags over time (re-orders); the rail focuses on
// what the user is actively brewing right now.
func newestOpenBag(bags []Bag) *BagSummary {
	var best *Bag
	for i := range bags {
		b := &bags[i]
		if b.FinishedAt != nil {
			continue
		}
		if best == nil || b.OpenedAt.After(best.OpenedAt) {
			best = b
		}
	}
	if best == nil {
		return nil
	}
	return &BagSummary{
		BagID:     best.BagID,
		OpenedAt:  best.OpenedAt,
		RoastDate: best.RoastDate,
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
