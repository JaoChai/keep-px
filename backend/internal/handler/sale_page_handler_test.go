package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
)

// ---------------------------------------------------------------------------
// Test setup
// ---------------------------------------------------------------------------

type salePageTestEnv struct {
	pixelRepo    *MockPixelRepo
	customerRepo *MockCustomerRepo
	subRepo      *MockSubscriptionRepo
	usageRepo    *MockEventUsageRepo
	salePageRepo *MockSalePageRepo
	creditRepo   *MockReplayCreditRepo
	handler      *SalePageHandler
	router       http.Handler
}

func setupSalePageTest(t *testing.T) *salePageTestEnv {
	t.Helper()

	pixelRepo := &MockPixelRepo{}
	customerRepo := &MockCustomerRepo{}
	subRepo := &MockSubscriptionRepo{}
	usageRepo := &MockEventUsageRepo{}
	salePageRepo := &MockSalePageRepo{}
	creditRepo := &MockReplayCreditRepo{}

	quotaService := newTestQuotaService(creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo, customerRepo)
	salePageService := newTestSalePageService(salePageRepo, customerRepo, pixelRepo, quotaService)
	h := NewSalePageHandler(salePageService, "https://test.keepx.io", testLogger())

	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testJWTSecret))
	r.Get("/sale-pages", h.List)
	r.Post("/sale-pages", h.Create)
	r.Put("/sale-pages/{id}", h.Update)
	r.Delete("/sale-pages/{id}", h.Delete)

	return &salePageTestEnv{
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

// setupQuotaAllowed configures mocks so sale page creation quota passes.
func (env *salePageTestEnv) setupQuotaAllowed(customerID string, currentSalePageCount int) {
	env.customerRepo.On("GetByID", mock.Anything, customerID).
		Return(&domain.Customer{ID: customerID, Plan: domain.PlanSandbox}, nil)
	env.subRepo.On("GetPixelSlotQuantity", mock.Anything, customerID).
		Return(0, nil)
	env.creditRepo.On("GetActiveByCustomerID", mock.Anything, customerID).
		Return([]*domain.ReplayCredit{}, nil)
	env.usageRepo.On("GetCurrentMonth", mock.Anything, customerID).
		Return((*domain.EventUsage)(nil), nil)
	env.salePageRepo.On("CountByCustomerID", mock.Anything, customerID).
		Return(currentSalePageCount, nil)
}

// setupQuotaExceeded configures mocks so sale page creation quota is exceeded (sandbox = 1 max).
func (env *salePageTestEnv) setupQuotaExceeded(customerID string) {
	env.setupQuotaAllowed(customerID, 1) // sandbox allows 1, count=1 means exceeded
}

// validSimpleContent returns minimal valid v1 SimpleContent JSON for tests.
func validSimpleContent() json.RawMessage {
	return json.RawMessage(`{
		"hero": {"title": "Test", "subtitle": "Sub", "image_url": ""},
		"body": {"description": "Body text", "features": [], "images": []},
		"cta": {"button_text": "Buy Now", "button_link": "https://example.com"},
		"contact": {"line_id": "", "phone": "", "website_url": ""},
		"tracking": {"cta_event_name": "Purchase", "content_name": "Test Product", "content_value": 100, "currency": "THB"},
		"style": {}
	}`)
}

// ---------------------------------------------------------------------------
// Tests: List
// ---------------------------------------------------------------------------

func TestSalePageHandler_List(t *testing.T) {
	t.Run("success returns sale page list", func(t *testing.T) {
		env := setupSalePageTest(t)
		customerID := testCustomerID
		token := testJWT(customerID, false)

		env.salePageRepo.On("ListByCustomerID", mock.Anything, customerID, 20, 0).
			Return([]*domain.SalePage{
				{
					ID:           testSalePageID,
					CustomerID:   customerID,
					Name:         "Page 1",
					Slug:         "page-1",
					TemplateName: "simple",
					Content:      validSimpleContent(),
				},
			}, 1, nil)

		rec := doRequest(env.router, http.MethodGet, "/sale-pages", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
		assert.Equal(t, 1, resp.Total)
	})
}

// ---------------------------------------------------------------------------
// Tests: Create
// ---------------------------------------------------------------------------

func TestSalePageHandler_Create(t *testing.T) {
	t.Run("success creates sale page", func(t *testing.T) {
		env := setupSalePageTest(t)
		customerID := testCustomerID
		token := testJWT(customerID, false)

		env.setupQuotaAllowed(customerID, 0)
		env.salePageRepo.On("SlugExists", mock.Anything, "my-page").Return(false, nil)
		env.salePageRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(nil)

		body := map[string]interface{}{
			"name":          "My Sale Page",
			"slug":          "my-page",
			"template_name": "simple",
			"content":       json.RawMessage(validSimpleContent()),
		}

		rec := doRequest(env.router, http.MethodPost, "/sale-pages", body, token)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
		assert.Empty(t, resp.Error)
	})

	t.Run("quota exceeded returns 402", func(t *testing.T) {
		env := setupSalePageTest(t)
		customerID := testCustomerID
		token := testJWT(customerID, false)

		env.setupQuotaExceeded(customerID)

		body := map[string]interface{}{
			"name":          "My Sale Page",
			"slug":          "my-page",
			"template_name": "simple",
			"content":       json.RawMessage(validSimpleContent()),
		}

		rec := doRequest(env.router, http.MethodPost, "/sale-pages", body, token)

		assert.Equal(t, http.StatusPaymentRequired, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "sale page limit exceeded")
	})

	t.Run("invalid slug returns 400", func(t *testing.T) {
		env := setupSalePageTest(t)
		customerID := testCustomerID
		token := testJWT(customerID, false)

		env.setupQuotaAllowed(customerID, 0)

		body := map[string]interface{}{
			"name":          "My Sale Page",
			"slug":          "INVALID SLUG!!",
			"template_name": "simple",
			"content":       json.RawMessage(validSimpleContent()),
		}

		rec := doRequest(env.router, http.MethodPost, "/sale-pages", body, token)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "slug")
	})

	t.Run("slug taken returns 409", func(t *testing.T) {
		env := setupSalePageTest(t)
		customerID := testCustomerID
		token := testJWT(customerID, false)

		env.setupQuotaAllowed(customerID, 0)
		env.salePageRepo.On("SlugExists", mock.Anything, "taken-slug").Return(true, nil)

		body := map[string]interface{}{
			"name":          "My Sale Page",
			"slug":          "taken-slug",
			"template_name": "simple",
			"content":       json.RawMessage(validSimpleContent()),
		}

		rec := doRequest(env.router, http.MethodPost, "/sale-pages", body, token)

		assert.Equal(t, http.StatusConflict, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "slug is already taken")
	})
}

// ---------------------------------------------------------------------------
// Tests: Update
// ---------------------------------------------------------------------------

func TestSalePageHandler_Update(t *testing.T) {
	t.Run("success updates sale page", func(t *testing.T) {
		env := setupSalePageTest(t)
		customerID := testCustomerID
		pageID := testSalePageID
		token := testJWT(customerID, false)

		existing := &domain.SalePage{
			ID:           pageID,
			CustomerID:   customerID,
			Name:         "Old Name",
			Slug:         "old-slug",
			TemplateName: "simple",
			Content:      validSimpleContent(),
		}

		env.salePageRepo.On("GetByID", mock.Anything, pageID).Return(existing, nil)
		env.salePageRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(nil)

		body := map[string]interface{}{
			"name": "Updated Name",
		}

		rec := doRequest(env.router, http.MethodPut, "/sale-pages/"+pageID, body, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
		assert.Empty(t, resp.Error)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupSalePageTest(t)
		customerID := testCustomerID
		pageID := "sp-nonexistent"
		token := testJWT(customerID, false)

		env.salePageRepo.On("GetByID", mock.Anything, pageID).Return((*domain.SalePage)(nil), nil)

		body := map[string]interface{}{
			"name": "Updated",
		}

		rec := doRequest(env.router, http.MethodPut, "/sale-pages/"+pageID, body, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "sale page not found")
	})

	t.Run("not owned returns 403", func(t *testing.T) {
		env := setupSalePageTest(t)
		customerID := testCustomerID
		otherCustomer := "cust-other"
		pageID := testSalePageID
		token := testJWT(customerID, false)

		env.salePageRepo.On("GetByID", mock.Anything, pageID).
			Return(&domain.SalePage{
				ID:           pageID,
				CustomerID:   otherCustomer,
				Name:         "Other's Page",
				Slug:         "other-page",
				TemplateName: "simple",
				Content:      validSimpleContent(),
			}, nil)

		body := map[string]interface{}{
			"name": "Hijack",
		}

		rec := doRequest(env.router, http.MethodPut, "/sale-pages/"+pageID, body, token)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "not owned")
	})
}

// ---------------------------------------------------------------------------
// Tests: Delete
// ---------------------------------------------------------------------------

func TestSalePageHandler_Delete(t *testing.T) {
	t.Run("success deletes sale page", func(t *testing.T) {
		env := setupSalePageTest(t)
		customerID := testCustomerID
		pageID := testSalePageID
		token := testJWT(customerID, false)

		env.salePageRepo.On("GetByID", mock.Anything, pageID).
			Return(&domain.SalePage{
				ID:           pageID,
				CustomerID:   customerID,
				Name:         "My Page",
				Slug:         "my-page",
				TemplateName: "simple",
				Content:      validSimpleContent(),
			}, nil)
		env.salePageRepo.On("Delete", mock.Anything, pageID).Return(nil)

		rec := doRequest(env.router, http.MethodDelete, "/sale-pages/"+pageID, nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Message, "sale page deleted")
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupSalePageTest(t)
		customerID := testCustomerID
		pageID := "sp-nonexistent"
		token := testJWT(customerID, false)

		env.salePageRepo.On("GetByID", mock.Anything, pageID).Return((*domain.SalePage)(nil), nil)

		rec := doRequest(env.router, http.MethodDelete, "/sale-pages/"+pageID, nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
