package handler

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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
			if tt.setupMocks != nil {
				tt.setupMocks(customerRepo)
			}

			authService := newTestAuthService(customerRepo)
			h := NewAuthHandler(authService, testLogger(), testAuth.KeyFunc(), testAuthIssuer)

			r := chi.NewRouter()
			r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
			if tt.setupMocks != nil {
				tt.setupMocks(customerRepo)
			}

			authService := newTestAuthService(customerRepo)
			h := NewAuthHandler(authService, testLogger(), testAuth.KeyFunc(), testAuthIssuer)

			r := chi.NewRouter()
			r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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

// ---------------------------------------------------------------------------
// TestAuthHandler_Session
//
// Session ตรวจ JWT เอง (ไม่ใช้ middleware JWTAuth) เพราะต้องสร้าง/ผูก customer
// ก่อน — middleware ตัวเดิมจะปฏิเสธด้วย customer_not_provisioned ก่อนจะทัน
// เคส "token ถูกต้อง + emailVerified:true → 200" เป็นเคสที่พิสูจน์ชื่อ claim
// "emailVerified" (camelCase) — ถ้าใครแก้เป็น "email_verified" จะดึงได้ false
// เสมอ → ErrEmailNotVerified → 403 แทน 200 → เทสต์นี้แดงทันที
// ---------------------------------------------------------------------------

func TestAuthHandler_Session(t *testing.T) {
	// newSessionHandler สร้าง handler จริง (real AuthService) ผูกกับ mock repo
	// ตามรูปแบบเดียวกับ handler test อื่น — AuthService เป็น concrete struct
	// จึง mock ไม่ได้ ต้องใช้ real service + mock repo
	newSessionHandler := func(t *testing.T, customerRepo *mocks.MockCustomerRepo) (http.Handler, *mocks.MockCustomerRepo) {
		t.Helper()
		h := NewAuthHandler(
			newTestAuthService(customerRepo),
			testLogger(), testAuth.KeyFunc(), testAuthIssuer,
		)
		r := chi.NewRouter()
		r.Post("/auth/session", h.Session)
		return r, customerRepo
	}

	t.Run("token ถูกต้อง + email ยืนยันแล้ว → 200 พร้อม customer", func(t *testing.T) {
		customerRepo := &mocks.MockCustomerRepo{}
		customerRepo.On("GetByAuthUserID", mock.Anything, "neon-1").Return(nil, nil)
		customerRepo.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil)
		customerRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			args.Get(1).(*domain.Customer).ID = "cust-new"
		})
		r, _ := newSessionHandler(t, customerRepo)

		token := testAuth.MintNeonToken(jwt.MapClaims{
			"sub": "neon-1", "email": "new@example.com", "name": "คนใหม่", "emailVerified": true,
		})
		rec := doRequest(r, "POST", "/auth/session", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Data domain.Customer `json:"data"`
		}
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "new@example.com", resp.Data.Email)
		assert.Equal(t, "คนใหม่", resp.Data.Name)
		customerRepo.AssertExpectations(t)
	})

	t.Run("ไม่มี Authorization header → 401", func(t *testing.T) {
		customerRepo := &mocks.MockCustomerRepo{}
		r, _ := newSessionHandler(t, customerRepo)

		rec := doRequest(r, "POST", "/auth/session", nil, "")

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("token เซ็นผิด → 401", func(t *testing.T) {
		customerRepo := &mocks.MockCustomerRepo{}
		r, _ := newSessionHandler(t, customerRepo)

		rec := doRequest(r, "POST", "/auth/session", nil, "not-a-real-token")

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("claims[emailVerified] เป็น false → ส่ง EmailVerified:false ให้ service → 403", func(t *testing.T) {
		customerRepo := &mocks.MockCustomerRepo{}
		customerRepo.On("GetByAuthUserID", mock.Anything, "neon-x").Return(nil, nil)
		// ไม่ตั้ง GetByEmail — service ต้องคืน ErrEmailNotVerified ก่อนถึง

		r, _ := newSessionHandler(t, customerRepo)
		token := testAuth.MintNeonToken(jwt.MapClaims{
			"sub": "neon-x", "email": "x@example.com", "emailVerified": false,
		})
		rec := doRequest(r, "POST", "/auth/session", nil, token)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		customerRepo.AssertExpectations(t)
	})

	t.Run("ProvisionCustomer คืน ErrAccountSuspended → 403", func(t *testing.T) {
		customerRepo := &mocks.MockCustomerRepo{}
		suspended := time.Now()
		customerRepo.On("GetByAuthUserID", mock.Anything, "neon-s").Return(&domain.Customer{
			ID: "cust-s", Email: "s@example.com", SuspendedAt: &suspended,
		}, nil)

		r, _ := newSessionHandler(t, customerRepo)
		token := testAuth.MintNeonToken(jwt.MapClaims{
			"sub": "neon-s", "email": "s@example.com", "emailVerified": true,
		})
		rec := doRequest(r, "POST", "/auth/session", nil, token)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		customerRepo.AssertExpectations(t)
	})
}
