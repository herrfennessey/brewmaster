package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/herrfennessey/brewmaster/api/internal/auth"
	"github.com/herrfennessey/brewmaster/api/internal/brew"
	"github.com/herrfennessey/brewmaster/api/internal/models"
	"github.com/herrfennessey/brewmaster/api/internal/store"
)

// requireUID extracts the authenticated uid set by auth.Middleware. The
// middleware always populates it on protected routes, so a miss here means
// the route was wired without auth — fail closed.
func requireUID(w http.ResponseWriter, r *http.Request) (string, bool) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user context")
		return "", false
	}
	return uid, true
}

// CoffeesHandler exposes user-scoped coffee CRUD on top of the storage repo.
type CoffeesHandler struct {
	repo store.Repo
	now  func() time.Time
}

// NewCoffeesHandler constructs the handler. now defaults to time.Now (UTC).
func NewCoffeesHandler(repo store.Repo) *CoffeesHandler {
	return &CoffeesHandler{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

type upsertRequest struct {
	Rating *int    `json:"rating,omitempty"`
	Notes  *string `json:"notes,omitempty"`
	Bag    struct {
		RoastDate string `json:"roast_date"`
	} `json:"bag"`
	BeanProfile models.BeanProfile `json:"bean_profile"`
}

type upsertResponse struct {
	CoffeeID string       `json:"coffee_id"`
	Coffee   store.Coffee `json:"coffee"`
	IsNew    bool         `json:"is_new"`
}

// maxNotesBytes caps server-side note storage. Without a cap the unbounded
// notes field is the easiest way for a single coffee doc to approach
// Firestore's 1 MiB document limit.
const maxNotesBytes = 4096

// validateRatingNotes enforces the data integrity rules used by both Upsert
// and Patch. Returns a non-empty error message on rejection.
func validateRatingNotes(rating *int, notes *string) string {
	if rating != nil && (*rating < 1 || *rating > 5) {
		return "rating must be between 1 and 5"
	}
	if notes != nil && len(*notes) > maxNotesBytes {
		return "notes exceeds 4 KiB limit"
	}
	return ""
}

// Upsert creates or merges a coffee record from a parsed bean.
func (h *CoffeesHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req upsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	key, ok := brew.CanonicalKey(&req.BeanProfile.Parsed)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity,
			"bean profile lacks the roaster + bean fields needed to dedupe")
		return
	}
	if msg := validateRatingNotes(req.Rating, req.Notes); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	in := store.UpsertInput{
		BeanProfile: req.BeanProfile,
		RoastDate:   req.Bag.RoastDate,
		Rating:      req.Rating,
		Notes:       req.Notes,
	}
	sum, coffee, created, err := h.repo.UpsertCoffee(r.Context(), uid, key, &in, h.now())
	if err != nil {
		slog.ErrorContext(r.Context(), "upsert coffee failed", "error", err, "uid", uid)
		writeError(w, http.StatusInternalServerError, "failed to save coffee")
		return
	}
	writeJSON(w, http.StatusOK, upsertResponse{
		CoffeeID: sum.CoffeeID,
		Coffee:   coffee,
		IsNew:    created,
	})
}

type listResponse struct {
	Coffees []store.CoffeeSummary `json:"coffees"`
}

// List returns the user's coffees, newest activity first.
func (h *CoffeesHandler) List(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUID(w, r)
	if !ok {
		return
	}
	coffees, err := h.repo.ListCoffees(r.Context(), uid)
	if err != nil {
		slog.ErrorContext(r.Context(), "list coffees failed", "error", err, "uid", uid)
		writeError(w, http.StatusInternalServerError, "failed to list coffees")
		return
	}
	writeJSON(w, http.StatusOK, listResponse{Coffees: coffees})
}

// Get returns one coffee by id (which is its canonical key).
func (h *CoffeesHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUID(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing coffee id")
		return
	}
	coffee, err := h.repo.GetCoffee(r.Context(), uid, id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "coffee not found")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "get coffee failed", "error", err, "uid", uid, "id", id)
		writeError(w, http.StatusInternalServerError, "failed to load coffee")
		return
	}
	writeJSON(w, http.StatusOK, coffee)
}

type patchRequest struct {
	Rating *int     `json:"rating,omitempty"`
	Notes  *string  `json:"notes,omitempty"`
	Clear  []string `json:"clear,omitempty"`
}

// patchClearable lists the fields a Patch request may explicitly clear via the
// `clear` array. Anything not in this set is rejected so a typo'd field name
// can't silently no-op or, worse, target an unintended Firestore path.
var patchClearable = map[string]bool{
	"rating": true,
	"notes":  true,
}

// Patch updates rating and/or notes on a coffee.
func (h *CoffeesHandler) Patch(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUID(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing coffee id")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var req patchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := validateRatingNotes(req.Rating, req.Notes); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	for _, f := range req.Clear {
		if !patchClearable[f] {
			writeError(w, http.StatusBadRequest, "clear: unknown field "+f)
			return
		}
	}
	coffee, err := h.repo.PatchCoffee(r.Context(), uid, id,
		store.PatchInput{Rating: req.Rating, Notes: req.Notes, Clear: req.Clear}, h.now())
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "coffee not found")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "patch coffee failed", "error", err, "uid", uid, "id", id)
		writeError(w, http.StatusInternalServerError, "failed to update coffee")
		return
	}
	writeJSON(w, http.StatusOK, coffee)
}

// Delete removes a coffee. Returns 204 on success, 404 when the coffee is
// already gone (or never belonged to this user — same response shape so we
// don't leak the existence of other users' coffees).
func (h *CoffeesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUID(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing coffee id")
		return
	}
	err := h.repo.DeleteCoffee(r.Context(), uid, id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "coffee not found")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "delete coffee failed", "error", err, "uid", uid, "id", id)
		writeError(w, http.StatusInternalServerError, "failed to delete coffee")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type lookupResponse struct {
	Coffee *store.CoffeeSummary `json:"coffee"`
}

// Lookup is the post-parse re-recognition probe. The client passes the
// canonical key returned by parse-bean; if a saved coffee matches, the client
// can route directly to it instead of running through review.
func (h *CoffeesHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUID(w, r)
	if !ok {
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "missing key parameter")
		return
	}
	c, found, err := h.repo.LookupBySlug(r.Context(), uid, key)
	if err != nil {
		slog.ErrorContext(r.Context(), "lookup coffee failed", "error", err, "uid", uid)
		writeError(w, http.StatusInternalServerError, "failed to look up coffee")
		return
	}
	resp := lookupResponse{}
	if found {
		resp.Coffee = &c
	}
	writeJSON(w, http.StatusOK, resp)
}
