package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{
			name:     "CF-Connecting-IP header takes priority",
			headers:  map[string]string{"CF-Connecting-IP": "1.2.3.4"},
			remote:   "10.0.0.1:12345",
			expected: "1.2.3.4",
		},
		{
			name:     "True-Client-IP fallback when no CF header",
			headers:  map[string]string{"True-Client-IP": "5.6.7.8"},
			remote:   "10.0.0.1:12345",
			expected: "5.6.7.8",
		},
		{
			name:     "CF-Connecting-IP wins over True-Client-IP",
			headers:  map[string]string{"CF-Connecting-IP": "1.2.3.4", "True-Client-IP": "5.6.7.8"},
			remote:   "10.0.0.1:12345",
			expected: "1.2.3.4",
		},
		{
			name:     "RemoteAddr with port stripped",
			headers:  map[string]string{},
			remote:   "192.168.1.1:54321",
			expected: "192.168.1.1",
		},
		{
			name:     "RemoteAddr without port returned as-is",
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
			name:     "invalid header value falls back to RemoteAddr",
			headers:  map[string]string{"CF-Connecting-IP": "not-an-ip"},
			remote:   "10.0.0.1:12345",
			expected: "10.0.0.1",
		},
		{
			name:     "header with leading/trailing whitespace trimmed",
			headers:  map[string]string{"CF-Connecting-IP": " 1.2.3.4 "},
			remote:   "10.0.0.1:12345",
			expected: "1.2.3.4",
		},
		{
			name:     "comma-separated list in header falls back to RemoteAddr",
			headers:  map[string]string{"CF-Connecting-IP": "1.2.3.4, 10.0.0.1"},
			remote:   "172.16.0.1:8080",
			expected: "172.16.0.1",
		},
		{
			name:     "empty header value ignored",
			headers:  map[string]string{"CF-Connecting-IP": "", "True-Client-IP": ""},
			remote:   "10.0.0.1:12345",
			expected: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest("POST", "/api/v1/events/ingest", nil)
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			r.RemoteAddr = tt.remote

			got := extractClientIP(r)
			assert.Equal(t, tt.expected, got)
		})
	}
}
