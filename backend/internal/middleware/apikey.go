package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/jaochai/pixlinks/backend/internal/repository"
)

type apiKeyCacheEntry struct {
	customerID string
	expiry     time.Time
}

func APIKeyAuthWithContext(ctx context.Context, customerRepo repository.CustomerRepository) func(http.Handler) http.Handler {
	var cache sync.Map
	const cacheTTL = 5 * time.Minute

	// Cleanup expired entries every 5 minutes; stops when ctx is done.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cache.Range(func(key, val any) bool {
					entry := val.(apiKeyCacheEntry)
					if time.Now().After(entry.expiry) {
						cache.Delete(key)
					}
					return true
				})
			}
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing API key")
				return
			}

			// Check cache
			if val, ok := cache.Load(apiKey); ok {
				entry := val.(apiKeyCacheEntry)
				if time.Now().Before(entry.expiry) {
					ctx := context.WithValue(r.Context(), CustomerIDKey, entry.customerID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				cache.Delete(apiKey)
			}

			customer, err := customerRepo.GetByAPIKey(r.Context(), apiKey)
			if err != nil || customer == nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid API key")
				return
			}

			cache.Store(apiKey, apiKeyCacheEntry{
				customerID: customer.ID,
				expiry:     time.Now().Add(cacheTTL),
			})

			ctx := context.WithValue(r.Context(), CustomerIDKey, customer.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
