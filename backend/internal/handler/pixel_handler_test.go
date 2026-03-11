package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

// httpResponseRecorder is an alias used across handler test files.
type httpResponseRecorder = httptest.ResponseRecorder

// ---------------------------------------------------------------------------
// Test setup
// ---------------------------------------------------------------------------

type pixelTestEnv struct {
	pixelRepo    *MockPixelRepo
	customerRepo *MockCustomerRepo
	subRepo      *MockSubscriptionRepo
	usageRepo    *MockEventUsageRepo
	salePageRepo *MockSalePageRepo
	creditRepo   *MockReplayCreditRepo
	handler      *PixelHandler
	router       http.Handler
}

func setupPixelTest(t *testing.T) *pixelTestEnv {
	t.Helper()

	pixelRepo := &MockPixelRepo{}
	customerRepo := &MockCustomerRepo{}
	subRepo := &MockSubscriptionRepo{}
	usageRepo := &MockEventUsageRepo{}
	salePageRepo := &MockSalePageRepo{}
	creditRepo := &MockReplayCreditRepo{}

	quotaService := newTestQuotaService(creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo, customerRepo)
	pixelService := newTestPixelService(pixelRepo, quotaService)
	h := NewPixelHandler(pixelService, testLogger())

	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testJWTSecret))
	r.Get("/pixels", h.List)
	r.Post("/pixels", h.Create)
	r.Put("/pixels/{id}", h.Update)
	r.Delete("/pixels/{id}", h.Delete)
	r.Post("/pixels/{id}/test", h.Test)

	return &pixelTestEnv{
		pixelRepo:    pixelRepo,
		customerRepo: customerRepo,
		subRepo:      subRepo,
		usageRepo:    usageRepo,
		salePageRepo: salePageRepo,
		creditRepo:   creditRepo,
		handler:      h,
		router:       r,
	}
}

// setupQuotaAllowed configures the standard mock calls for CheckPixelCreationQuota
// to pass for a sandbox customer with `currentCount` existing pixels.
func (env *pixelTestEnv) setupQuotaAllowed(customerID string, currentCount int) {
	env.customerRepo.On("GetByID", mock.Anything, customerID).
		Return(&domain.Customer{ID: customerID, Plan: domain.PlanSandbox}, nil)
	env.subRepo.On("GetActiveByCustomerID", mock.Anything, customerID).
		Return([]*domain.Subscription{}, nil)
	env.creditRepo.On("GetActiveByCustomerID", mock.Anything, customerID).
		Return([]*domain.ReplayCredit{}, nil)
	env.usageRepo.On("GetCurrentMonth", mock.Anything, customerID).
		Return((*domain.EventUsage)(nil), nil)
	env.pixelRepo.On("CountByCustomerID", mock.Anything, customerID).
		Return(currentCount, nil)
}

// setupQuotaExceeded configures mocks so pixel creation quota is exceeded (sandbox = 2 max).
func (env *pixelTestEnv) setupQuotaExceeded(customerID string) {
	env.setupQuotaAllowed(customerID, 2) // sandbox allows 2, so count=2 means exceeded
}

// ---------------------------------------------------------------------------
// Tests: List
// ---------------------------------------------------------------------------

func TestPixelHandler_List(t *testing.T) {
	t.Run("success returns pixel list", func(t *testing.T) {
		env := setupPixelTest(t)
		customerID := testCustomerID
		token := testJWT(customerID, false)

		env.pixelRepo.On("ListByCustomerID", mock.Anything, customerID).
			Return([]*domain.Pixel{
				{ID: testPixelID, CustomerID: customerID, Name: "Pixel 1", FBPixelID: "111"},
				{ID: "px-2", CustomerID: customerID, Name: "Pixel 2", FBPixelID: "222"},
			}, nil)

		rec := doRequest(env.router, http.MethodGet, "/pixels", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
		pixels, ok := resp.Data.([]interface{})
		require.True(t, ok)
		assert.Len(t, pixels, 2)
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		env := setupPixelTest(t)
		rec := doRequest(env.router, http.MethodGet, "/pixels", nil, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: Create
// ---------------------------------------------------------------------------

func TestPixelHandler_Create(t *testing.T) {
	t.Run("success creates pixel", func(t *testing.T) {
		env := setupPixelTest(t)
		customerID := testCustomerID
		token := testJWT(customerID, false)

		env.setupQuotaAllowed(customerID, 0)
		env.pixelRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Pixel")).
			Return(nil)

		body := service.CreatePixelInput{
			FBPixelID:     "123456",
			FBAccessToken: "token-abc",
			Name:          "My Pixel",
		}

		rec := doRequest(env.router, http.MethodPost, "/pixels", body, token)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
		assert.Empty(t, resp.Error)
	})

	t.Run("missing required fields returns 400", func(t *testing.T) {
		env := setupPixelTest(t)
		customerID := testCustomerID
		token := testJWT(customerID, false)

		// Only provide name, missing fb_pixel_id and fb_access_token
		body := map[string]string{
			"name": "Missing pixel ID and token",
		}

		rec := doRequest(env.router, http.MethodPost, "/pixels", body, token)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotEmpty(t, resp.Error)
	})

	t.Run("quota exceeded returns 402", func(t *testing.T) {
		env := setupPixelTest(t)
		customerID := testCustomerID
		token := testJWT(customerID, false)

		env.setupQuotaExceeded(customerID)

		body := service.CreatePixelInput{
			FBPixelID:     "123456",
			FBAccessToken: "token-abc",
			Name:          "My Pixel",
		}

		rec := doRequest(env.router, http.MethodPost, "/pixels", body, token)

		assert.Equal(t, http.StatusPaymentRequired, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "pixel limit exceeded")
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		env := setupPixelTest(t)

		body := service.CreatePixelInput{
			FBPixelID:     "123456",
			FBAccessToken: "token-abc",
			Name:          "My Pixel",
		}

		rec := doRequest(env.router, http.MethodPost, "/pixels", body, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: Update
// ---------------------------------------------------------------------------

func TestPixelHandler_Update(t *testing.T) {
	t.Run("success updates pixel", func(t *testing.T) {
		env := setupPixelTest(t)
		customerID := testCustomerID
		pixelID := testPixelID
		token := testJWT(customerID, false)

		existingPixel := &domain.Pixel{
			ID:            pixelID,
			CustomerID:    customerID,
			FBPixelID:     "111",
			FBAccessToken: "old-token",
			Name:          "Old Name",
			IsActive:      true,
			Status:        "active",
		}

		env.pixelRepo.On("GetByID", mock.Anything, pixelID).Return(existingPixel, nil)
		env.pixelRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Pixel")).Return(nil)

		newName := "Updated Pixel"
		body := service.UpdatePixelInput{
			Name: &newName,
		}

		rec := doRequest(env.router, http.MethodPut, "/pixels/"+pixelID, body, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
		assert.Empty(t, resp.Error)
	})

	t.Run("pixel not found returns 404", func(t *testing.T) {
		env := setupPixelTest(t)
		customerID := testCustomerID
		pixelID := testPixelMissing
		token := testJWT(customerID, false)

		env.pixelRepo.On("GetByID", mock.Anything, pixelID).Return((*domain.Pixel)(nil), nil)

		newName := "Updated"
		body := service.UpdatePixelInput{Name: &newName}

		rec := doRequest(env.router, http.MethodPut, "/pixels/"+pixelID, body, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "pixel not found")
	})

	t.Run("not owned returns 403", func(t *testing.T) {
		env := setupPixelTest(t)
		customerID := testCustomerID
		otherCustomer := "cust-other"
		pixelID := testPixelID
		token := testJWT(customerID, false)

		env.pixelRepo.On("GetByID", mock.Anything, pixelID).
			Return(&domain.Pixel{
				ID:         pixelID,
				CustomerID: otherCustomer,
				Name:       "Other's Pixel",
			}, nil)

		newName := "Hijack"
		body := service.UpdatePixelInput{Name: &newName}

		rec := doRequest(env.router, http.MethodPut, "/pixels/"+pixelID, body, token)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "not owned")
	})

	t.Run("backup pixel is self returns 400", func(t *testing.T) {
		env := setupPixelTest(t)
		customerID := testCustomerID
		pixelID := testPixelID
		token := testJWT(customerID, false)

		env.pixelRepo.On("GetByID", mock.Anything, pixelID).
			Return(&domain.Pixel{
				ID:            pixelID,
				CustomerID:    customerID,
				FBPixelID:     "111",
				FBAccessToken: "token",
				Name:          "My Pixel",
				IsActive:      true,
				Status:        "active",
			}, nil)

		selfRef := pixelID
		body := service.UpdatePixelInput{
			BackupPixelID: &selfRef,
		}

		rec := doRequest(env.router, http.MethodPut, "/pixels/"+pixelID, body, token)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "its own backup")
	})
}

// ---------------------------------------------------------------------------
// Tests: Delete
// ---------------------------------------------------------------------------

func TestPixelHandler_Delete(t *testing.T) {
	t.Run("success deletes pixel", func(t *testing.T) {
		env := setupPixelTest(t)
		customerID := testCustomerID
		pixelID := testPixelID
		token := testJWT(customerID, false)

		env.pixelRepo.On("GetByID", mock.Anything, pixelID).
			Return(&domain.Pixel{
				ID:         pixelID,
				CustomerID: customerID,
				Name:       "My Pixel",
			}, nil)
		env.pixelRepo.On("Delete", mock.Anything, pixelID).Return(nil)

		rec := doRequest(env.router, http.MethodDelete, "/pixels/"+pixelID, nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Message, "pixel deleted")
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupPixelTest(t)
		customerID := testCustomerID
		pixelID := testPixelMissing
		token := testJWT(customerID, false)

		env.pixelRepo.On("GetByID", mock.Anything, pixelID).Return((*domain.Pixel)(nil), nil)

		rec := doRequest(env.router, http.MethodDelete, "/pixels/"+pixelID, nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: Test (pixel connection test)
// ---------------------------------------------------------------------------

func TestPixelHandler_Test(t *testing.T) {
	t.Run("no access token returns 400", func(t *testing.T) {
		env := setupPixelTest(t)
		customerID := testCustomerID
		pixelID := testPixelID
		token := testJWT(customerID, false)

		env.pixelRepo.On("GetByID", mock.Anything, pixelID).
			Return(&domain.Pixel{
				ID:            pixelID,
				CustomerID:    customerID,
				FBPixelID:     "111",
				FBAccessToken: "", // no access token
				Name:          "My Pixel",
			}, nil)

		rec := doRequest(env.router, http.MethodPost, "/pixels/"+pixelID+"/test", nil, token)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "no access token")
	})

	t.Run("pixel not found returns 404", func(t *testing.T) {
		env := setupPixelTest(t)
		customerID := testCustomerID
		pixelID := testPixelMissing
		token := testJWT(customerID, false)

		env.pixelRepo.On("GetByID", mock.Anything, pixelID).Return((*domain.Pixel)(nil), nil)

		rec := doRequest(env.router, http.MethodPost, "/pixels/"+pixelID+"/test", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
