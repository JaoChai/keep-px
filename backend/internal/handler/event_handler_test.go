package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"

	"github.com/jaochai/pixlinks/backend/internal/middleware"
)

// resolvedClientIP drives r through the same ClientIPFromHeader("X-Real-IP")
// middleware that router.go registers, then reports what middleware.ClientIP
// observes — the path a real request takes through the chain.
func resolvedClientIP(r *http.Request) string {
	var got string
	h := chimiddleware.ClientIPFromHeader("X-Real-IP")(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		got = middleware.ClientIP(req)
	}))
	h.ServeHTTP(httptest.NewRecorder(), r)
	return got
}

func TestResolvedClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{
			name:     "client IP resolved from X-Real-IP via middleware",
			headers:  map[string]string{"X-Real-IP": "203.0.113.10"},
			remote:   "10.0.0.1:12345",
			expected: "203.0.113.10",
		},
		{
			name:     "forged CF-Connecting-IP ignored, falls back to RemoteAddr",
			headers:  map[string]string{"CF-Connecting-IP": "9.9.9.9"},
			remote:   "10.0.0.1:12345",
			expected: "10.0.0.1",
		},
		{
			name:     "forged True-Client-IP ignored, falls back to RemoteAddr",
			headers:  map[string]string{"True-Client-IP": "5.6.7.8"},
			remote:   "10.0.0.2:12345",
			expected: "10.0.0.2",
		},
		{
			name:     "no headers falls back to RemoteAddr",
			headers:  map[string]string{},
			remote:   "192.168.1.1:54321",
			expected: "192.168.1.1",
		},
		{
			name:     "RemoteAddr without port returned verbatim",
			headers:  map[string]string{},
			remote:   "192.168.1.1",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv6 RemoteAddr with port stripped",
			headers:  map[string]string{},
			remote:   "[::1]:8080",
			expected: "::1",
		},
		{
			name:     "forged headers cannot override resolved X-Real-IP",
			headers:  map[string]string{"X-Real-IP": "203.0.113.20", "CF-Connecting-IP": "9.9.9.9", "True-Client-IP": "5.6.7.8"},
			remote:   "10.0.0.3:12345",
			expected: "203.0.113.20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest("POST", "/api/v1/events/ingest", nil)
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			r.RemoteAddr = tt.remote

			got := resolvedClientIP(r)
			assert.Equal(t, tt.expected, got)
		})
	}
}
