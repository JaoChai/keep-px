package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCustomerRepo is a minimal mock that implements CustomerRepository.
// Only GetByAPIKey is used by the API key middleware; all other methods panic.
type mockCustomerRepo struct {
	repository.CustomerRepository
	getByAPIKeyFn func(ctx context.Context, apiKey string) (*domain.Customer, error)
	callCount     atomic.Int64
}

func (m *mockCustomerRepo) GetByAPIKey(ctx context.Context, apiKey string) (*domain.Customer, error) {
	m.callCount.Add(1)
	return m.getByAPIKeyFn(ctx, apiKey)
}

func TestAPIKeyAuth_MissingAPIKey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := &mockCustomerRepo{
		getByAPIKeyFn: func(_ context.Context, _ string) (*domain.Customer, error) {
			t.Fatal("repo should not be called")
			return nil, nil
		},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := APIKeyAuthWithContext(ctx, repo)(next)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/ingest", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "missing API key", body["error"])
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := &mockCustomerRepo{
		getByAPIKeyFn: func(_ context.Context, _ string) (*domain.Customer, error) {
			return nil, nil // not found
		},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := APIKeyAuthWithContext(ctx, repo)(next)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/ingest", nil)
	req.Header.Set("X-API-Key", "invalid-key-12345")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid API key", body["error"])
}

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := &mockCustomerRepo{
		getByAPIKeyFn: func(_ context.Context, apiKey string) (*domain.Customer, error) {
			if apiKey == "valid-api-key" {
				return &domain.Customer{
					ID:     "customer-456",
					APIKey: "valid-api-key",
				}, nil
			}
			return nil, nil
		},
	}

	var gotCustomerID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCustomerID = GetCustomerID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := APIKeyAuthWithContext(ctx, repo)(next)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/ingest", nil)
	req.Header.Set("X-API-Key", "valid-api-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "customer-456", gotCustomerID)
}

func TestAPIKeyAuth_CacheHit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := &mockCustomerRepo{
		getByAPIKeyFn: func(_ context.Context, apiKey string) (*domain.Customer, error) {
			return &domain.Customer{
				ID:     "customer-789",
				APIKey: apiKey,
			}, nil
		},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := APIKeyAuthWithContext(ctx, repo)(next)

	// First request — should hit the repo
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/events/ingest", nil)
	req1.Header.Set("X-API-Key", "cached-key")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)
	assert.Equal(t, int64(1), repo.callCount.Load())

	// Second request with the same key — should use cache, NOT call repo again
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/events/ingest", nil)
	req2.Header.Set("X-API-Key", "cached-key")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, int64(1), repo.callCount.Load(), "repo should not be called again for a cached key")
}

func TestAPIKeyAuth_CacheExpiry(t *testing.T) {
	// We cannot easily test the 5-minute TTL without mocking time,
	// but we can verify that a different key does hit the repo.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := &mockCustomerRepo{
		getByAPIKeyFn: func(_ context.Context, apiKey string) (*domain.Customer, error) {
			return &domain.Customer{
				ID:     "customer-" + apiKey,
				APIKey: apiKey,
			}, nil
		},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := APIKeyAuthWithContext(ctx, repo)(next)

	// First key
	req1 := httptest.NewRequest(http.MethodPost, "/", nil)
	req1.Header.Set("X-API-Key", "key-a")
	handler.ServeHTTP(httptest.NewRecorder(), req1)

	// Different key — should call repo
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.Header.Set("X-API-Key", "key-b")
	handler.ServeHTTP(httptest.NewRecorder(), req2)

	assert.Equal(t, int64(2), repo.callCount.Load(), "different keys should each call repo")
}

func TestAPIKeyAuth_RepoError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := &mockCustomerRepo{
		getByAPIKeyFn: func(_ context.Context, _ string) (*domain.Customer, error) {
			return nil, assert.AnError
		},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := APIKeyAuthWithContext(ctx, repo)(next)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-API-Key", "some-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid API key", body["error"])
}

func TestAPIKeyAuth_ContextCancellation(t *testing.T) {
	// Verify that the background cleanup goroutine stops when context is cancelled.
	// We just ensure no panic/leak by cancelling immediately.
	ctx, cancel := context.WithCancel(context.Background())

	repo := &mockCustomerRepo{
		getByAPIKeyFn: func(_ context.Context, _ string) (*domain.Customer, error) {
			return &domain.Customer{ID: "c-1"}, nil
		},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := APIKeyAuthWithContext(ctx, repo)(next)
	cancel()

	// Allow the goroutine to exit
	time.Sleep(10 * time.Millisecond)

	// Middleware should still work (cache lookup / repo call)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-API-Key", "key-after-cancel")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
