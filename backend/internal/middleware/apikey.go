package middleware

import (
	"context"
	"net/http"

	"github.com/jaochai/pixlinks/backend/internal/repository"
)

func APIKeyAuth(customerRepo repository.CustomerRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				http.Error(w, `{"error":"missing API key"}`, http.StatusUnauthorized)
				return
			}

			customer, err := customerRepo.GetByAPIKey(r.Context(), apiKey)
			if err != nil || customer == nil {
				http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), CustomerIDKey, customer.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
