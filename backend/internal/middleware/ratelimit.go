package middleware

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimitWithContext returns a per-IP rate limiting middleware that stops
// its cleanup goroutine when ctx is cancelled.
func RateLimitWithContext(ctx context.Context, rps int) func(http.Handler) http.Handler {
	var (
		mu       sync.RWMutex
		visitors = make(map[string]*visitor)
	)

	// Cleanup stale entries every 5 minutes; stops when ctx is done.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				for ip, v := range visitors {
					if time.Since(v.lastSeen) > 10*time.Minute {
						delete(visitors, ip)
					}
				}
				mu.Unlock()
			}
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			mu.RLock()
			v, exists := visitors[ip]
			mu.RUnlock()

			if !exists {
				mu.Lock()
				// Double-check after acquiring write lock
				v, exists = visitors[ip]
				if !exists {
					v = &visitor{limiter: rate.NewLimiter(rate.Limit(rps), rps*2)}
					visitors[ip] = v
				}
				mu.Unlock()
			}

			mu.RLock()
			v.lastSeen = time.Now()
			mu.RUnlock()

			if !v.limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit returns a per-IP rate limiting middleware.
// Deprecated: prefer RateLimitWithContext to avoid goroutine leaks.
func RateLimit(rps int) func(http.Handler) http.Handler {
	return RateLimitWithContext(context.Background(), rps)
}
