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

func APIKeyAuth(customerRepo repository.CustomerRepository) func(http.Handler) http.Handler {
	var cache sync.Map
	const cacheTTL = 5 * time.Minute

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				http.Error(w, `{"error":"missing API key"}`, http.StatusUnauthorized)
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
				http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
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
