package middleware

import (
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

// RateLimit returns a per-IP rate limiting middleware.
func RateLimit(rps int) func(http.Handler) http.Handler {
	var (
		mu       sync.Mutex
		visitors = make(map[string]*visitor)
	)

	// Cleanup stale entries every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 10*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			mu.Lock()
			v, exists := visitors[ip]
			if !exists {
				v = &visitor{limiter: rate.NewLimiter(rate.Limit(rps), rps*2)}
				visitors[ip] = v
			}
			v.lastSeen = time.Now()
			mu.Unlock()

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
