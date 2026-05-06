package auth_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	firebaseauth "firebase.google.com/go/v4/auth"

	"github.com/herrfennessey/brewmaster/api/internal/auth"
)

type stubVerifier struct {
	err error
	uid string
}

func (s *stubVerifier) VerifyIDToken(_ context.Context, _ string) (*firebaseauth.Token, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &firebaseauth.Token{UID: s.uid}, nil
}

func makeRequest(authHeader string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	return req
}

func sinkHandler(captured *string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, _ := auth.UserIDFromContext(r.Context())
		*captured = uid
		w.WriteHeader(http.StatusNoContent)
	})
}

func TestMiddleware_DisabledInjectsLocalDev(t *testing.T) {
	var got string
	mw := auth.Middleware(nil)(sinkHandler(&got))

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, makeRequest(""))

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	if got != auth.LocalDevUserID {
		t.Errorf("expected uid %q, got %q", auth.LocalDevUserID, got)
	}
}

func TestMiddleware_AcceptsValidToken(t *testing.T) {
	var got string
	mw := auth.Middleware(&stubVerifier{uid: "user-123"})(sinkHandler(&got))

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, makeRequest("Bearer tok"))

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	if got != "user-123" {
		t.Errorf("expected uid user-123, got %q", got)
	}
}

func TestMiddleware_RejectsMissingHeader(t *testing.T) {
	var got string
	mw := auth.Middleware(&stubVerifier{uid: "user-123"})(sinkHandler(&got))

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, makeRequest(""))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if got != "" {
		t.Errorf("handler should not run, got uid %q", got)
	}
}

func TestMiddleware_RejectsMalformedHeader(t *testing.T) {
	mw := auth.Middleware(&stubVerifier{uid: "user-123"})(sinkHandler(new(string)))

	cases := []string{"abc", "Bearer", "Bearer ", "Token abc"}
	for _, h := range cases {
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, makeRequest(h))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("header %q: expected 401, got %d", h, rec.Code)
		}
	}
}

func TestMiddleware_RejectsInvalidToken(t *testing.T) {
	mw := auth.Middleware(&stubVerifier{err: errors.New("expired")})(sinkHandler(new(string)))

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, makeRequest("Bearer bad"))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !contains(string(body), "invalid id token") {
		t.Errorf("expected error body to mention invalid id token, got %s", string(body))
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
