package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminOnly_IsAdmin(t *testing.T) {
	var called bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := AdminOnly(next)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	ctx := context.WithValue(req.Context(), IsAdminKey, true)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "next handler should be called for admin")
}

func TestAdminOnly_NotAdmin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for non-admin")
	})

	handler := AdminOnly(next)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	ctx := context.WithValue(req.Context(), IsAdminKey, false)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "forbidden", body["error"])
}

func TestAdminOnly_NotSetInContext(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called when is_admin is not in context")
	})

	handler := AdminOnly(next)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	// No IsAdminKey in context at all
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "forbidden", body["error"])
}

func TestGetIsAdmin_Values(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected bool
	}{
		{
			name: "true in context",
			setup: func() context.Context {
				return context.WithValue(context.Background(), IsAdminKey, true)
			},
			expected: true,
		},
		{
			name: "false in context",
			setup: func() context.Context {
				return context.WithValue(context.Background(), IsAdminKey, false)
			},
			expected: false,
		},
		{
			name: "not set in context",
			setup: func() context.Context {
				return context.Background()
			},
			expected: false,
		},
		{
			name: "wrong type in context",
			setup: func() context.Context {
				return context.WithValue(context.Background(), IsAdminKey, "true")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			assert.Equal(t, tt.expected, GetIsAdmin(ctx))
		})
	}
}
