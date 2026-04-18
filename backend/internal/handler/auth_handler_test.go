package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/repository/mocks"
)

// ---------------------------------------------------------------------------
// TestAuthHandler_Me
// ---------------------------------------------------------------------------

func TestAuthHandler_Me(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		setupMocks func(*mocks.MockCustomerRepo)
		wantStatus int
		wantError  string
	}{
		{
			name:  "success — valid token returns customer data",
			token: testJWT(testCustomerID, false),
			setupMocks: func(cr *mocks.MockCustomerRepo) {
				cr.On("GetByID", mock.Anything, testCustomerID).Return(&domain.Customer{
					ID:    testCustomerID,
					Email: "user@example.com",
					Name:  "Test User",
					Plan:  domain.PlanSandbox,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "no auth header — 401",
			token:      "",
			wantStatus: http.StatusUnauthorized,
			wantError:  "missing authorization header",
		},
		{
			name:       "invalid token — 401",
			token:      "totally-invalid-token",
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid token",
		},
		{
			name:  "customer not found — 401",
			token: testJWT("nonexistent", false),
			setupMocks: func(cr *mocks.MockCustomerRepo) {
				cr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customerRepo := &mocks.MockCustomerRepo{}
			refreshTokenRepo := &mocks.MockRefreshTokenRepo{}
			if tt.setupMocks != nil {
				tt.setupMocks(customerRepo)
			}

			authService := newTestAuthService(customerRepo, refreshTokenRepo)
			h := NewAuthHandler(authService, testConfig(), testLogger())

			r := chi.NewRouter()
			r.Use(middleware.JWTAuth(testJWTSecret))
			r.Get("/auth/me", h.Me)

			rec := doRequest(r, "GET", "/auth/me", nil, tt.token)
			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantError != "" {
				var resp APIResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Contains(t, resp.Error, tt.wantError)
			}

			if tt.wantStatus == http.StatusOK {
				var resp APIResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.NotNil(t, resp.Data)
			}

			customerRepo.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthHandler_Logout
// ---------------------------------------------------------------------------

func TestAuthHandler_Logout(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		setupMocks func(*mocks.MockRefreshTokenRepo)
		wantStatus int
		wantMsg    string
		wantError  string
	}{
		{
			name:  "success — logged out",
			token: testJWT(testCustomerID, false),
			setupMocks: func(rt *mocks.MockRefreshTokenRepo) {
				rt.On("DeleteByCustomerID", mock.Anything, testCustomerID).Return(nil)
			},
			wantStatus: http.StatusOK,
			wantMsg:    "logged out",
		},
		{
			name:       "no auth — 401",
			token:      "",
			wantStatus: http.StatusUnauthorized,
			wantError:  "missing authorization header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customerRepo := &mocks.MockCustomerRepo{}
			refreshTokenRepo := &mocks.MockRefreshTokenRepo{}
			if tt.setupMocks != nil {
				tt.setupMocks(refreshTokenRepo)
			}

			authService := newTestAuthService(customerRepo, refreshTokenRepo)
			h := NewAuthHandler(authService, testConfig(), testLogger())

			r := chi.NewRouter()
			r.Use(middleware.JWTAuth(testJWTSecret))
			r.Post("/auth/logout", h.Logout)

			rec := doRequest(r, "POST", "/auth/logout", nil, tt.token)
			assert.Equal(t, tt.wantStatus, rec.Code)

			var resp APIResponse
			err := json.NewDecoder(rec.Body).Decode(&resp)
			assert.NoError(t, err)

			if tt.wantMsg != "" {
				assert.Equal(t, tt.wantMsg, resp.Message)
			}
			if tt.wantError != "" {
				assert.Contains(t, resp.Error, tt.wantError)
			}

			refreshTokenRepo.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthHandler_Refresh
// ---------------------------------------------------------------------------

func TestAuthHandler_Refresh(t *testing.T) {
	tests := []struct {
		name       string
		body       interface{}
		setupMocks func(*mocks.MockCustomerRepo, *mocks.MockRefreshTokenRepo)
		wantStatus int
		wantError  string
	}{
		{
			name: "success — returns new tokens",
			body: map[string]string{"refresh_token": "valid-refresh-token"},
			setupMocks: func(cr *mocks.MockCustomerRepo, rt *mocks.MockRefreshTokenRepo) {
				// The service hashes the token before lookup.
				rt.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(
					testCustomerID,
					time.Now().Add(7*24*time.Hour),
					nil,
				)
				rt.On("DeleteByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(nil)
				cr.On("GetByID", mock.Anything, testCustomerID).Return(&domain.Customer{
					ID:    testCustomerID,
					Email: "user@example.com",
					Name:  "Test User",
					Plan:  domain.PlanSandbox,
				}, nil)
				// New refresh token is created during token generation.
				rt.On("Create", mock.Anything, testCustomerID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid refresh token — 401",
			body: map[string]string{"refresh_token": "bad-token"},
			setupMocks: func(cr *mocks.MockCustomerRepo, rt *mocks.MockRefreshTokenRepo) {
				rt.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(
					"",
					time.Time{},
					nil,
				)
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid refresh token",
		},
		{
			name:       "missing refresh token — 400",
			body:       map[string]string{},
			wantStatus: http.StatusBadRequest,
			wantError:  "refresh_token is required",
		},
		{
			name: "suspended account — 403",
			body: map[string]string{"refresh_token": "suspended-user-token"},
			setupMocks: func(cr *mocks.MockCustomerRepo, rt *mocks.MockRefreshTokenRepo) {
				rt.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(
					"cust-suspended",
					time.Now().Add(7*24*time.Hour),
					nil,
				)
				rt.On("DeleteByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(nil)
				suspendedAt := time.Now().Add(-24 * time.Hour)
				cr.On("GetByID", mock.Anything, "cust-suspended").Return(&domain.Customer{
					ID:          "cust-suspended",
					Email:       "suspended@example.com",
					Name:        "Suspended User",
					SuspendedAt: &suspendedAt,
				}, nil)
			},
			wantStatus: http.StatusForbidden,
			wantError:  "account suspended",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customerRepo := &mocks.MockCustomerRepo{}
			refreshTokenRepo := &mocks.MockRefreshTokenRepo{}
			if tt.setupMocks != nil {
				tt.setupMocks(customerRepo, refreshTokenRepo)
			}

			authService := newTestAuthService(customerRepo, refreshTokenRepo)
			h := NewAuthHandler(authService, testConfig(), testLogger())

			// Refresh is a public endpoint — no JWT middleware.
			r := chi.NewRouter()
			r.Post("/auth/refresh", h.Refresh)

			rec := doRequest(r, "POST", "/auth/refresh", tt.body, "")
			assert.Equal(t, tt.wantStatus, rec.Code)

			var resp APIResponse
			err := json.NewDecoder(rec.Body).Decode(&resp)
			assert.NoError(t, err)

			if tt.wantError != "" {
				assert.Contains(t, resp.Error, tt.wantError)
			}

			if tt.wantStatus == http.StatusOK {
				assert.NotNil(t, resp.Data)
				dataMap, ok := resp.Data.(map[string]interface{})
				assert.True(t, ok, "data should be a map")
				assert.NotEmpty(t, dataMap["access_token"])
				assert.NotEmpty(t, dataMap["refresh_token"])
			}

			customerRepo.AssertExpectations(t)
			refreshTokenRepo.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthHandler_GoogleAuthCallback
// ---------------------------------------------------------------------------

func TestAuthHandler_GoogleAuthCallback(t *testing.T) {
	tests := []struct {
		name            string
		formValue       map[string]string
		csrfCookie      string
		wantStatusCode  int
		wantRedirectURL string
	}{
		{
			name:            "missing credential — redirects to /login?error=auth_failed",
			formValue:       map[string]string{},
			wantStatusCode:  http.StatusFound,
			wantRedirectURL: "http://localhost:5173/login?error=auth_failed",
		},
		{
			name:            "missing csrf token in form — redirects to /login?error=auth_failed",
			formValue:       map[string]string{"credential": "test-cred"},
			csrfCookie:      "abc123",
			wantStatusCode:  http.StatusFound,
			wantRedirectURL: "http://localhost:5173/login?error=auth_failed",
		},
		{
			name:            "csrf token mismatch — redirects to /login?error=auth_failed",
			formValue:       map[string]string{"credential": "test-cred", "g_csrf_token": "xyz789"},
			csrfCookie:      "abc123",
			wantStatusCode:  http.StatusFound,
			wantRedirectURL: "http://localhost:5173/login?error=auth_failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customerRepo := &mocks.MockCustomerRepo{}
			refreshTokenRepo := &mocks.MockRefreshTokenRepo{}

			authService := newTestAuthService(customerRepo, refreshTokenRepo)
			h := NewAuthHandler(authService, testConfig(), testLogger())

			// Create request with form data
			body := ""
			for k, v := range tt.formValue {
				if body != "" {
					body += "&"
				}
				body += k + "=" + url.QueryEscape(v)
			}

			req := httptest.NewRequest("POST", "/api/v1/auth/google/callback", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Set CSRF cookie if provided
			if tt.csrfCookie != "" {
				req.AddCookie(&http.Cookie{
					Name:  "g_csrf_token",
					Value: tt.csrfCookie,
				})
			}

			rec := httptest.NewRecorder()
			h.GoogleAuthCallback(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
			location := rec.Header().Get("Location")
			assert.Equal(t, tt.wantRedirectURL, location)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthHandler_RegenerateAPIKey
// ---------------------------------------------------------------------------

func TestAuthHandler_RegenerateAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		setupMocks func(*mocks.MockCustomerRepo)
		wantStatus int
		wantError  string
	}{
		{
			name:  "success — returns updated customer",
			token: testJWT(testCustomerID, false),
			setupMocks: func(cr *mocks.MockCustomerRepo) {
				cr.On("RegenerateAPIKey", mock.Anything, testCustomerID, mock.AnythingOfType("string")).Return(&domain.Customer{
					ID:     testCustomerID,
					Email:  "user@example.com",
					Name:   "Test User",
					APIKey: "pk_new_generated_key",
					Plan:   domain.PlanSandbox,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "no auth — 401",
			token:      "",
			wantStatus: http.StatusUnauthorized,
			wantError:  "missing authorization header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customerRepo := &mocks.MockCustomerRepo{}
			refreshTokenRepo := &mocks.MockRefreshTokenRepo{}
			if tt.setupMocks != nil {
				tt.setupMocks(customerRepo)
			}

			authService := newTestAuthService(customerRepo, refreshTokenRepo)
			h := NewAuthHandler(authService, testConfig(), testLogger())

			r := chi.NewRouter()
			r.Use(middleware.JWTAuth(testJWTSecret))
			r.Post("/auth/regenerate-api-key", h.RegenerateAPIKey)

			rec := doRequest(r, "POST", "/auth/regenerate-api-key", nil, tt.token)
			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantError != "" {
				var resp APIResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Contains(t, resp.Error, tt.wantError)
			}

			if tt.wantStatus == http.StatusOK {
				var resp APIResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.NotNil(t, resp.Data)
			}

			customerRepo.AssertExpectations(t)
		})
	}
}
