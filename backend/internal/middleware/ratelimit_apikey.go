package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type apiKeyCtxKey struct{}

const apiKeyLimiterTTL = 1 * time.Hour

type apiKeyLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type apiKeyLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*apiKeyLimiter
	rps      rate.Limit
	burst    int
}

func newAPIKeyLimiterStore(rps int, burst int) *apiKeyLimiterStore {
	s := &apiKeyLimiterStore{
		limiters: make(map[string]*apiKeyLimiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
	go s.gcLoop()
	return s
}

func (s *apiKeyLimiterStore) get(key string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l, ok := s.limiters[key]; ok {
		l.lastSeen = time.Now()
		return l.limiter
	}
	l := &apiKeyLimiter{
		limiter:  rate.NewLimiter(s.rps, s.burst),
		lastSeen: time.Now(),
	}
	s.limiters[key] = l
	return l.limiter
}

func (s *apiKeyLimiterStore) gcLoop() {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for range t.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-apiKeyLimiterTTL)
		for k, l := range s.limiters {
			if l.lastSeen.Before(cutoff) {
				delete(s.limiters, k)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimitByAPIKey rate-limits by API key from request context.
// Requests without an API key pass through unchanged.
func RateLimitByAPIKey(rps int, burst int) func(http.Handler) http.Handler {
	store := newAPIKeyLimiterStore(rps, burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, ok := r.Context().Value(apiKeyCtxKey{}).(string)
			if !ok || key == "" {
				next.ServeHTTP(w, r)
				return
			}
			if !store.get(key).Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
