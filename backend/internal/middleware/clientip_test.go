package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// rateLimitChain wires ClientIPFromHeader before RateLimitWithContext, matching
// the order used in router.go, so these tests exercise the same client-IP
// resolution path that production requests follow.
func rateLimitChain(rps int) (http.Handler, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := chimiddleware.ClientIPFromHeader("X-Real-IP")(RateLimitWithContext(ctx, rps)(final))
	return h, cancel
}

func TestRateLimitKey_IgnoresForgedTrueClientIP(t *testing.T) {
	h, cancel := rateLimitChain(1)
	defer cancel()

	forgedTrueClientIPs := []string{
		"203.0.113.10", "203.0.113.11", "203.0.113.12", "203.0.113.13", "203.0.113.14",
	}
	var blocked int
	for _, forged := range forgedTrueClientIPs {
		req := httptest.NewRequest(http.MethodGet, "/p/slug", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		req.Header.Set("True-Client-IP", forged)   // client-controlled, must be ignored
		req.Header.Set("X-Real-IP", "203.0.113.1") // trusted header, identical every request
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			blocked++
		}
	}
	if blocked == 0 {
		t.Fatal("expected at least one 429: forging True-Client-IP must not buy fresh buckets")
	}
}

func TestRateLimitKey_IgnoresForgedXForwardedFor(t *testing.T) {
	h, cancel := rateLimitChain(1)
	defer cancel()

	forgedXFF := []string{
		"198.51.100.10", "198.51.100.11", "198.51.100.12", "198.51.100.13", "198.51.100.14",
	}
	var blocked int
	for _, forged := range forgedXFF {
		req := httptest.NewRequest(http.MethodGet, "/p/slug", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		req.Header.Set("X-Forwarded-For", forged) // client-controlled, must be ignored
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			blocked++
		}
	}
	if blocked == 0 {
		t.Fatal("expected at least one 429: forging X-Forwarded-For must not buy fresh buckets")
	}
}

func TestRateLimitKey_SeparatesDistinctXRealIP(t *testing.T) {
	h, cancel := rateLimitChain(1)
	defer cancel()

	distinctRealIPs := []string{
		"198.51.100.20", "198.51.100.21", "198.51.100.22", "198.51.100.23", "198.51.100.24",
	}
	for _, ip := range distinctRealIPs {
		req := httptest.NewRequest(http.MethodGet, "/p/slug", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		req.Header.Set("X-Real-IP", ip)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("distinct X-Real-IP %s: want 200, got %d (genuine clients must keep separate buckets)", ip, w.Code)
		}
	}
}

func TestRateLimitKey_FallsBackToRemoteAddr(t *testing.T) {
	h, cancel := rateLimitChain(1)
	defer cancel()

	var blocked int
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/p/slug", nil)
		req.RemoteAddr = "10.0.0.9:4321"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			blocked++
		}
	}
	if blocked == 0 {
		t.Fatal("expected at least one 429 from a single RemoteAddr with no proxy headers")
	}
}
