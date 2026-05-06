// Package auth verifies Firebase ID tokens and exposes the resulting user id
// to downstream handlers via request context.
package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
)

// LocalDevUserID is injected when DISABLE_AUTH=true so dev workflows do not
// require a real Firebase token.
const LocalDevUserID = "local-dev"

type ctxKey int

const userIDKey ctxKey = iota

// Verifier is the subset of *auth.Client that the middleware depends on.
// Defining it as an interface keeps the middleware unit-testable without the
// Firebase Admin SDK.
type Verifier interface {
	VerifyIDToken(ctx context.Context, token string) (*auth.Token, error)
}

// Middleware verifies the bearer token in the Authorization header and
// injects the uid into the request context. When verifier is nil (DISABLE_AUTH
// mode) every request is accepted as the LocalDevUserID.
func Middleware(verifier Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if verifier == nil {
				next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), LocalDevUserID)))
				return
			}

			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeUnauthorized(w, "missing or malformed Authorization header")
				return
			}

			tok, err := verifier.VerifyIDToken(r.Context(), token)
			if err != nil {
				slog.InfoContext(r.Context(), "auth: invalid id token", "error", err)
				writeUnauthorized(w, "invalid id token")
				return
			}

			next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), tok.UID)))
		})
	}
}

// UserIDFromContext returns the authenticated user id, or empty + false when
// no auth middleware ran on the request.
func UserIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(userIDKey).(string)
	return uid, ok && uid != ""
}

func withUserID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, userIDKey, uid)
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	if _, err := w.Write([]byte(`{"error":"` + msg + `"}`)); err != nil {
		slog.Error("auth: failed to write unauthorized response", "error", err)
	}
}
