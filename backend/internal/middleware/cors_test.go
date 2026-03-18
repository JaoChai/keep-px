package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORS_AllowedOrigin(t *testing.T) {
	allowedOrigins := []string{"https://app.example.com", "http://localhost:5173"}

	var called bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS(allowedOrigins)(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pixels", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called)
	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	allowedOrigins := []string{"https://app.example.com"}

	var called bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS(allowedOrigins)(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pixels", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "next handler should still be called for non-OPTIONS")
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"),
		"ACAO header should NOT be set for disallowed origin")
}

func TestCORS_PreflightOptions(t *testing.T) {
	allowedOrigins := []string{"https://app.example.com"}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for preflight")
	})

	handler := CORS(allowedOrigins)(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/pixels", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "X-API-Key")
	assert.Equal(t, "86400", rec.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_NonOptionsPassesThrough(t *testing.T) {
	allowedOrigins := []string{"https://app.example.com"}

	var called bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})

	handler := CORS(allowedOrigins)(next)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/ingest", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.True(t, called)
}

func TestCORS_NoOriginHeader(t *testing.T) {
	allowedOrigins := []string{"https://app.example.com"}

	var called bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS(allowedOrigins)(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pixels", nil)
	// No Origin header
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"),
		"ACAO header should not be set when no Origin is provided")
}

func TestCORS_MultipleOrigins(t *testing.T) {
	allowedOrigins := []string{"https://app.example.com", "http://localhost:5173"}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS(allowedOrigins)(next)

	tests := []struct {
		name          string
		origin        string
		expectACAO    string
		expectACAOSet bool
	}{
		{"first origin", "https://app.example.com", "https://app.example.com", true},
		{"second origin", "http://localhost:5173", "http://localhost:5173", true},
		{"unknown origin", "https://other.com", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if tt.expectACAOSet {
				assert.Equal(t, tt.expectACAO, rec.Header().Get("Access-Control-Allow-Origin"))
			} else {
				assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestCORS_PreflightDisallowedOrigin(t *testing.T) {
	allowedOrigins := []string{"https://app.example.com"}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for OPTIONS")
	})

	handler := CORS(allowedOrigins)(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/pixels", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"),
		"ACAO header should not be set for disallowed origin even on preflight")
}
