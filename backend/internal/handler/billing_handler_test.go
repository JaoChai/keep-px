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
		subRepo.On("GetActiveByCustomerID", mock.Anything, testCustomerID).Return([]*domain.Subscription{}, nil)
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
	t.Run("with pack_type returns 503 when Stripe not configured", func(t *testing.T) {
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

		// When Stripe is not configured, CreateReplayPackCheckout calls EnsureStripeCustomer
		// which returns ErrStripeNotConfigured.
		customer := &domain.Customer{ID: testCustomerID, Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/checkout", h.CreateCheckout)

		body := map[string]string{"pack_type": domain.PackReplay1}
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT(testCustomerID, false))

		assert.Equal(t, 503, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "billing is not configured", resp.Error)
	})

	t.Run("with addon_type returns 503 when Stripe not configured", func(t *testing.T) {
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

		// addonPriceID returns error for unknown addon type when price ID is empty
		// For a known addon type with no price, it still returns the empty string which fails.
		// The addon type validation happens at the service level.
		customer := &domain.Customer{ID: testCustomerID, Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/checkout", h.CreateCheckout)

		body := map[string]string{"addon_type": domain.AddonEvents1M}
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT(testCustomerID, false))

		// Should be 503 (Stripe not configured) because EnsureStripeCustomer is called
		// OR 400 (invalid addon type) if the price ID is empty.
		// With empty config, the addon price ID will be empty string, which means
		// addonPriceID returns the empty price ID (not an error) since the key exists.
		// Then EnsureStripeCustomer returns ErrStripeNotConfigured.
		assert.Equal(t, 503, rec.Code)
	})

	t.Run("with plan_type returns 400 for invalid plan when price ID empty", func(t *testing.T) {
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

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/checkout", h.CreateCheckout)

		// plan_type with empty price ID in config returns ErrInvalidPlanType (400)
		body := map[string]string{"plan_type": domain.SubTypePlanLaunch}
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT(testCustomerID, false))

		assert.Equal(t, 400, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "invalid plan type", resp.Error)
	})

	t.Run("missing all types returns 400", func(t *testing.T) {
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
		assert.Equal(t, "plan_type, pack_type, or addon_type is required", resp.Error)
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

		body := map[string]string{"pack_type": domain.PackReplay1}
		rec := doRequest(r, "POST", "/billing/checkout", body, "")

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
