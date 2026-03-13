package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

// nbMockPool implements service.Pool for tests (Begin is never called in these tests).
type nbMockPool struct{}

func (m *nbMockPool) Begin(_ context.Context) (pgx.Tx, error) { return nil, nil }

// ---------------------------------------------------------------------------
// Billing handler test helpers
// ---------------------------------------------------------------------------

// nbBillingConfig returns a minimal config with no Stripe keys configured.
func nbBillingConfig() *config.Config {
	return &config.Config{
		JWTSecret:     testJWTSecret,
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
		// Stripe keys intentionally empty to test non-Stripe paths
	}
}

// nbNewTestBillingService creates a BillingService with mock repos and no Stripe config.
func nbNewTestBillingService(
	purchaseRepo *MockPurchaseRepo,
	creditRepo *MockReplayCreditRepo,
	subRepo *MockSubscriptionRepo,
	customerRepo *MockCustomerRepo,
	webhookRepo *MockWebhookEventRepo,
) *service.BillingService {
	return service.NewBillingService(
		purchaseRepo,
		creditRepo,
		subRepo,
		customerRepo,
		webhookRepo,
		&nbMockPool{},
		nbBillingConfig(),
	)
}

// nbNewTestQuotaService creates a QuotaService with mock repos.
func nbNewTestQuotaService(
	creditRepo *MockReplayCreditRepo,
	subRepo *MockSubscriptionRepo,
	usageRepo *MockEventUsageRepo,
	pixelRepo *MockPixelRepo,
	salePageRepo *MockSalePageRepo,
	customerRepo *MockCustomerRepo,
) *service.QuotaService {
	return service.NewQuotaService(creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo, customerRepo)
}

// ---------------------------------------------------------------------------
// GetBillingOverview tests
// ---------------------------------------------------------------------------

func TestBillingHandler_GetBillingOverview(t *testing.T) {
	t.Run("success returns 200 with billing data", func(t *testing.T) {
		purchaseRepo := &MockPurchaseRepo{}
		creditRepo := &MockReplayCreditRepo{}
		subRepo := &MockSubscriptionRepo{}
		customerRepo := &MockCustomerRepo{}
		webhookRepo := &MockWebhookEventRepo{}

		billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)

		// QuotaService repos (separate instances to avoid mock collisions)
		qCreditRepo := &MockReplayCreditRepo{}
		qSubRepo := &MockSubscriptionRepo{}
		qUsageRepo := &MockEventUsageRepo{}
		qPixelRepo := &MockPixelRepo{}
		qSalePageRepo := &MockSalePageRepo{}
		qCustomerRepo := &MockCustomerRepo{}

		quotaSvc := nbNewTestQuotaService(qCreditRepo, qSubRepo, qUsageRepo, qPixelRepo, qSalePageRepo, qCustomerRepo)

		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		customer := &domain.Customer{
			ID:    testCustomerID,
			Email: "test@example.com",
			Name:  "Test User",
			Plan:  domain.PlanSandbox,
		}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)
		purchaseRepo.On("ListByCustomerID", mock.Anything, testCustomerID).Return([]*domain.Purchase{}, nil)
		creditRepo.On("GetActiveByCustomerID", mock.Anything, testCustomerID).Return([]*domain.ReplayCredit{}, nil)
		subRepo.On("ListByCustomerID", mock.Anything, testCustomerID).Return([]*domain.Subscription{}, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Get("/billing/overview", h.GetBillingOverview)

		rec := doRequest(r, "GET", "/billing/overview", nil, testJWT(testCustomerID, false))

		assert.Equal(t, 200, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Data)

		customerRepo.AssertExpectations(t)
		purchaseRepo.AssertExpectations(t)
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		billingSvc := nbNewTestBillingService(
			&MockPurchaseRepo{}, &MockReplayCreditRepo{}, &MockSubscriptionRepo{},
			&MockCustomerRepo{}, &MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Get("/billing/overview", h.GetBillingOverview)

		rec := doRequest(r, "GET", "/billing/overview", nil, "")
		assert.Equal(t, 401, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// GetQuota tests
// ---------------------------------------------------------------------------

func TestBillingHandler_GetQuota(t *testing.T) {
	t.Run("success returns 200 with quota data", func(t *testing.T) {
		// BillingService repos (unused but needed for constructor)
		billingSvc := nbNewTestBillingService(
			&MockPurchaseRepo{}, &MockReplayCreditRepo{}, &MockSubscriptionRepo{},
			&MockCustomerRepo{}, &MockWebhookEventRepo{},
		)

		// QuotaService repos
		creditRepo := &MockReplayCreditRepo{}
		subRepo := &MockSubscriptionRepo{}
		usageRepo := &MockEventUsageRepo{}
		pixelRepo := &MockPixelRepo{}
		salePageRepo := &MockSalePageRepo{}
		customerRepo := &MockCustomerRepo{}

		quotaSvc := nbNewTestQuotaService(creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo, customerRepo)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		customer := &domain.Customer{
			ID:   testCustomerID,
			Plan: domain.PlanSandbox,
		}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)
		subRepo.On("GetPixelSlotQuantity", mock.Anything, testCustomerID).Return(0, nil)
		creditRepo.On("GetActiveByCustomerID", mock.Anything, testCustomerID).Return([]*domain.ReplayCredit{}, nil)
		usageRepo.On("GetCurrentMonth", mock.Anything, testCustomerID).Return((*domain.EventUsage)(nil), nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Get("/billing/quota", h.GetQuota)

		rec := doRequest(r, "GET", "/billing/quota", nil, testJWT(testCustomerID, false))

		assert.Equal(t, 200, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Data)

		dataMap, ok := resp.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, domain.PlanSandbox, dataMap["plan"])

		customerRepo.AssertExpectations(t)
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		billingSvc := nbNewTestBillingService(
			&MockPurchaseRepo{}, &MockReplayCreditRepo{}, &MockSubscriptionRepo{},
			&MockCustomerRepo{}, &MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Get("/billing/quota", h.GetQuota)

		rec := doRequest(r, "GET", "/billing/quota", nil, "")
		assert.Equal(t, 401, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// CreateCheckout tests
// ---------------------------------------------------------------------------

func TestBillingHandler_CreateCheckout(t *testing.T) {
	t.Run("with pixel_slots returns 503 when Stripe not configured", func(t *testing.T) {
		purchaseRepo := &MockPurchaseRepo{}
		creditRepo := &MockReplayCreditRepo{}
		subRepo := &MockSubscriptionRepo{}
		customerRepo := &MockCustomerRepo{}
		webhookRepo := &MockWebhookEventRepo{}

		billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		customer := &domain.Customer{ID: testCustomerID, Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/checkout", h.CreateCheckout)

		body := map[string]interface{}{"type": "pixel_slots", "quantity": 5}
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT(testCustomerID, false))

		assert.Equal(t, 503, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "billing is not configured", resp.Error)
	})

	t.Run("with replay_single returns 503 when Stripe not configured", func(t *testing.T) {
		purchaseRepo := &MockPurchaseRepo{}
		creditRepo := &MockReplayCreditRepo{}
		subRepo := &MockSubscriptionRepo{}
		customerRepo := &MockCustomerRepo{}
		webhookRepo := &MockWebhookEventRepo{}

		billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		customer := &domain.Customer{ID: testCustomerID, Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/checkout", h.CreateCheckout)

		body := map[string]string{"type": "replay_single"}
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT(testCustomerID, false))

		assert.Equal(t, 503, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "billing is not configured", resp.Error)
	})

	t.Run("with replay_monthly returns 503 when Stripe not configured", func(t *testing.T) {
		purchaseRepo := &MockPurchaseRepo{}
		creditRepo := &MockReplayCreditRepo{}
		subRepo := &MockSubscriptionRepo{}
		customerRepo := &MockCustomerRepo{}
		webhookRepo := &MockWebhookEventRepo{}

		billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		customer := &domain.Customer{ID: testCustomerID, Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/checkout", h.CreateCheckout)

		body := map[string]string{"type": "replay_monthly"}
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT(testCustomerID, false))

		assert.Equal(t, 503, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "billing is not configured", resp.Error)
	})

	t.Run("with invalid type returns 400", func(t *testing.T) {
		billingSvc := nbNewTestBillingService(
			&MockPurchaseRepo{}, &MockReplayCreditRepo{}, &MockSubscriptionRepo{},
			&MockCustomerRepo{}, &MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/checkout", h.CreateCheckout)

		body := map[string]string{"type": "invalid_type"}
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT(testCustomerID, false))

		assert.Equal(t, 400, rec.Code)
	})

	t.Run("missing type returns 400", func(t *testing.T) {
		billingSvc := nbNewTestBillingService(
			&MockPurchaseRepo{}, &MockReplayCreditRepo{}, &MockSubscriptionRepo{},
			&MockCustomerRepo{}, &MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/checkout", h.CreateCheckout)

		body := map[string]string{}
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT(testCustomerID, false))

		assert.Equal(t, 400, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "type is required", resp.Error)
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		billingSvc := nbNewTestBillingService(
			&MockPurchaseRepo{}, &MockReplayCreditRepo{}, &MockSubscriptionRepo{},
			&MockCustomerRepo{}, &MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/checkout", h.CreateCheckout)

		body := map[string]string{"type": "pixel_slots"}
		rec := doRequest(r, "POST", "/billing/checkout", body, "")

		assert.Equal(t, 401, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// UpdateSlots tests
// ---------------------------------------------------------------------------

func TestBillingHandler_UpdateSlots(t *testing.T) {
	t.Run("returns 503 when Stripe not configured", func(t *testing.T) {
		purchaseRepo := &MockPurchaseRepo{}
		creditRepo := &MockReplayCreditRepo{}
		subRepo := &MockSubscriptionRepo{}
		customerRepo := &MockCustomerRepo{}
		webhookRepo := &MockWebhookEventRepo{}

		billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		customer := &domain.Customer{ID: testCustomerID, Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Put("/billing/slots", h.UpdateSlots)

		body := map[string]int{"quantity": 3}
		rec := doRequest(r, "PUT", "/billing/slots", body, testJWT(testCustomerID, false))

		assert.Equal(t, 503, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "billing is not configured", resp.Error)
	})

	t.Run("invalid quantity returns 400", func(t *testing.T) {
		billingSvc := nbNewTestBillingService(
			&MockPurchaseRepo{}, &MockReplayCreditRepo{}, &MockSubscriptionRepo{},
			&MockCustomerRepo{}, &MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Put("/billing/slots", h.UpdateSlots)

		body := map[string]int{"quantity": 0}
		rec := doRequest(r, "PUT", "/billing/slots", body, testJWT(testCustomerID, false))

		assert.Equal(t, 400, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "quantity must be at least 1", resp.Error)
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		billingSvc := nbNewTestBillingService(
			&MockPurchaseRepo{}, &MockReplayCreditRepo{}, &MockSubscriptionRepo{},
			&MockCustomerRepo{}, &MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Put("/billing/slots", h.UpdateSlots)

		body := map[string]int{"quantity": 3}
		rec := doRequest(r, "PUT", "/billing/slots", body, "")

		assert.Equal(t, 401, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// CreatePortalSession tests
// ---------------------------------------------------------------------------

func TestBillingHandler_CreatePortalSession(t *testing.T) {
	t.Run("returns 503 when Stripe not configured", func(t *testing.T) {
		purchaseRepo := &MockPurchaseRepo{}
		creditRepo := &MockReplayCreditRepo{}
		subRepo := &MockSubscriptionRepo{}
		customerRepo := &MockCustomerRepo{}
		webhookRepo := &MockWebhookEventRepo{}

		billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		customer := &domain.Customer{ID: testCustomerID, Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/portal", h.CreatePortalSession)

		rec := doRequest(r, "POST", "/billing/portal", nil, testJWT(testCustomerID, false))

		assert.Equal(t, 503, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "billing is not configured", resp.Error)
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		billingSvc := nbNewTestBillingService(
			&MockPurchaseRepo{}, &MockReplayCreditRepo{}, &MockSubscriptionRepo{},
			&MockCustomerRepo{}, &MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&MockReplayCreditRepo{}, &MockSubscriptionRepo{}, &MockEventUsageRepo{},
			&MockPixelRepo{}, &MockSalePageRepo{}, &MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/portal", h.CreatePortalSession)

		rec := doRequest(r, "POST", "/billing/portal", nil, "")
		assert.Equal(t, 401, rec.Code)
	})
}
