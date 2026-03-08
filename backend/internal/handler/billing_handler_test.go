package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

// ---------------------------------------------------------------------------
// Mock Pool for BillingService (service.Pool interface)
// ---------------------------------------------------------------------------

// nbMockTx implements pgx.Tx for the mock pool (minimal stub).
type nbMockTx struct{ mock.Mock }

func (m *nbMockTx) Begin(ctx context.Context) (pgx.Tx, error)   { return nil, nil }
func (m *nbMockTx) Commit(ctx context.Context) error             { return nil }
func (m *nbMockTx) Rollback(ctx context.Context) error           { return nil }
func (m *nbMockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *nbMockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (m *nbMockTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (m *nbMockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *nbMockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *nbMockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *nbMockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { return nil }
func (m *nbMockTx) Conn() *pgx.Conn                                               { return nil }

// nbMockPool implements service.Pool for tests.
type nbMockPool struct{ mock.Mock }

func (m *nbMockPool) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(pgx.Tx), args.Error(1)
}

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
			ID:    "cust-1",
			Email: "test@example.com",
			Name:  "Test User",
			Plan:  domain.PlanSandbox,
		}
		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(customer, nil)
		purchaseRepo.On("ListByCustomerID", mock.Anything, "cust-1").Return([]*domain.Purchase{}, nil)
		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{}, nil)
		subRepo.On("ListByCustomerID", mock.Anything, "cust-1").Return([]*domain.Subscription{}, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Get("/billing/overview", h.GetBillingOverview)

		rec := doRequest(r, "GET", "/billing/overview", nil, testJWT("cust-1", false))

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
			ID:   "cust-1",
			Plan: domain.PlanSandbox,
		}
		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(customer, nil)
		subRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.Subscription{}, nil)
		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{}, nil)
		usageRepo.On("GetCurrentMonth", mock.Anything, "cust-1").Return((*domain.EventUsage)(nil), nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Get("/billing/quota", h.GetQuota)

		rec := doRequest(r, "GET", "/billing/quota", nil, testJWT("cust-1", false))

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
		customer := &domain.Customer{ID: "cust-1", Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/checkout", h.CreateCheckout)

		body := map[string]string{"pack_type": domain.PackReplay1}
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT("cust-1", false))

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
		customer := &domain.Customer{ID: "cust-1", Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/checkout", h.CreateCheckout)

		body := map[string]string{"addon_type": domain.AddonEvents1M}
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT("cust-1", false))

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
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT("cust-1", false))

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
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT("cust-1", false))

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

		customer := &domain.Customer{ID: "cust-1", Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Post("/billing/portal", h.CreatePortalSession)

		rec := doRequest(r, "POST", "/billing/portal", nil, testJWT("cust-1", false))

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
