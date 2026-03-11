package middleware

import (
	"context"
	"net/http"
	"time"
)

// DBTimeout wraps each request context with a deadline so that all
// downstream pgx queries automatically abort after the given duration.
func DBTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
