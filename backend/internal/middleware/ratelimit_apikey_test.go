package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitByAPIKey_AllowsUnderLimit(t *testing.T) {
	mw := RateLimitByAPIKey(5, 5)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req = req.WithContext(context.WithValue(req.Context(), apiKeyCtxKey{}, "key-a"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: want 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimitByAPIKey_BlocksWhenExceeded(t *testing.T) {
	mw := RateLimitByAPIKey(1, 1)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var blocked int
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req = req.WithContext(context.WithValue(req.Context(), apiKeyCtxKey{}, "key-b"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			blocked++
		}
	}
	if blocked == 0 {
		t.Fatal("expected at least one 429, got none")
	}
}

func TestRateLimitByAPIKey_SeparateBucketsPerKey(t *testing.T) {
	mw := RateLimitByAPIKey(1, 1)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, key := range []string{"key-c", "key-d"} {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req = req.WithContext(context.WithValue(req.Context(), apiKeyCtxKey{}, key))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("key %s: want 200, got %d", key, w.Code)
		}
	}
}

func TestRateLimitByAPIKey_PassthroughWhenNoKey(t *testing.T) {
	mw := RateLimitByAPIKey(1, 1)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("no key: want 200, got %d", w.Code)
	}
}
