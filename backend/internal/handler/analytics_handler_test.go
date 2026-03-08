package handler

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/jaochai/pixlinks/backend/internal/middleware"
)

// ---------------------------------------------------------------------------
// Analytics handler tests
//
// AnalyticsService requires *pgxpool.Pool directly (not repository interfaces),
// which cannot be mocked without a real database connection. These tests verify
// that the JWT auth middleware correctly rejects unauthenticated requests.
// ---------------------------------------------------------------------------

func TestAnalyticsHandler_Overview_NoAuth(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testJWTSecret))
	r.Get("/analytics/overview", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, http.StatusOK, APIResponse{Data: map[string]string{"status": "ok"}})
	})

	rec := doRequest(r, "GET", "/analytics/overview", nil, "")
	assert.Equal(t, 401, rec.Code)
}

func TestAnalyticsHandler_EventChart_NoAuth(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testJWTSecret))
	r.Get("/analytics/events/chart", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, http.StatusOK, APIResponse{Data: map[string]string{"status": "ok"}})
	})

	rec := doRequest(r, "GET", "/analytics/events/chart", nil, "")
	assert.Equal(t, 401, rec.Code)
}

func TestAnalyticsHandler_Overview_WithAuth(t *testing.T) {
	// Verify that a valid JWT passes through the middleware to the handler.
	// We use a stub handler since we can't construct AnalyticsService without pgxpool.
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testJWTSecret))
	r.Get("/analytics/overview", func(w http.ResponseWriter, r *http.Request) {
		customerID := middleware.GetCustomerID(r.Context())
		JSON(w, http.StatusOK, APIResponse{Data: map[string]string{"customer_id": customerID}})
	})

	rec := doRequest(r, "GET", "/analytics/overview", nil, testJWT("cust-1", false))
	assert.Equal(t, 200, rec.Code)
}

func TestAnalyticsHandler_EventChart_WithAuth(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testJWTSecret))
	r.Get("/analytics/events/chart", func(w http.ResponseWriter, r *http.Request) {
		customerID := middleware.GetCustomerID(r.Context())
		JSON(w, http.StatusOK, APIResponse{Data: map[string]string{"customer_id": customerID}})
	})

	rec := doRequest(r, "GET", "/analytics/events/chart", nil, testJWT("cust-1", false))
	assert.Equal(t, 200, rec.Code)
}
