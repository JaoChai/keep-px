package handler

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/stripe/stripe-go/v86/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/repository/mocks"
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

// nbWebhookSecret is the Stripe signing secret embedded in the webhook test
// handler config. Payloads are signed with webhook.ComputeSignature using it.
const nbWebhookSecret = "whsec_test_secret"

// nbBillingConfigWithWebhook returns nbBillingConfig with the webhook signing
// secret set, so Webhook can verify signed test payloads. Stripe API keys stay
// empty — the handler only needs the secret for ConstructEventWithOptions.
func nbBillingConfigWithWebhook() *config.Config {
	cfg := nbBillingConfig()
	cfg.StripeWebhookSecret = nbWebhookSecret
	return cfg
}

// nbNewTestBillingService creates a BillingService with mock repos and no Stripe config.
func nbNewTestBillingService(
	purchaseRepo *mocks.MockPurchaseRepo,
	creditRepo *mocks.MockReplayCreditRepo,
	subRepo *mocks.MockSubscriptionRepo,
	customerRepo *mocks.MockCustomerRepo,
	webhookRepo *mocks.MockWebhookEventRepo,
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
	creditRepo *mocks.MockReplayCreditRepo,
	subRepo *mocks.MockSubscriptionRepo,
	usageRepo *mocks.MockEventUsageRepo,
	pixelRepo *mocks.MockPixelRepo,
	salePageRepo *mocks.MockSalePageRepo,
	customerRepo *mocks.MockCustomerRepo,
) *service.QuotaService {
	return service.NewQuotaService(creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo, customerRepo)
}

// ---------------------------------------------------------------------------
// GetBillingOverview tests
// ---------------------------------------------------------------------------

func TestBillingHandler_GetBillingOverview(t *testing.T) {
	t.Run("success returns 200 with billing data", func(t *testing.T) {
		purchaseRepo := &mocks.MockPurchaseRepo{}
		creditRepo := &mocks.MockReplayCreditRepo{}
		subRepo := &mocks.MockSubscriptionRepo{}
		customerRepo := &mocks.MockCustomerRepo{}
		webhookRepo := &mocks.MockWebhookEventRepo{}

		billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)

		// QuotaService repos (separate instances to avoid mock collisions)
		qCreditRepo := &mocks.MockReplayCreditRepo{}
		qSubRepo := &mocks.MockSubscriptionRepo{}
		qUsageRepo := &mocks.MockEventUsageRepo{}
		qPixelRepo := &mocks.MockPixelRepo{}
		qSalePageRepo := &mocks.MockSalePageRepo{}
		qCustomerRepo := &mocks.MockCustomerRepo{}

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
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
			&mocks.MockPurchaseRepo{}, &mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{},
			&mocks.MockCustomerRepo{}, &mocks.MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
			&mocks.MockPurchaseRepo{}, &mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{},
			&mocks.MockCustomerRepo{}, &mocks.MockWebhookEventRepo{},
		)

		// QuotaService repos
		creditRepo := &mocks.MockReplayCreditRepo{}
		subRepo := &mocks.MockSubscriptionRepo{}
		usageRepo := &mocks.MockEventUsageRepo{}
		pixelRepo := &mocks.MockPixelRepo{}
		salePageRepo := &mocks.MockSalePageRepo{}
		customerRepo := &mocks.MockCustomerRepo{}

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
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
			&mocks.MockPurchaseRepo{}, &mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{},
			&mocks.MockCustomerRepo{}, &mocks.MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
		purchaseRepo := &mocks.MockPurchaseRepo{}
		creditRepo := &mocks.MockReplayCreditRepo{}
		subRepo := &mocks.MockSubscriptionRepo{}
		customerRepo := &mocks.MockCustomerRepo{}
		webhookRepo := &mocks.MockWebhookEventRepo{}

		billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		customer := &domain.Customer{ID: testCustomerID, Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
		purchaseRepo := &mocks.MockPurchaseRepo{}
		creditRepo := &mocks.MockReplayCreditRepo{}
		subRepo := &mocks.MockSubscriptionRepo{}
		customerRepo := &mocks.MockCustomerRepo{}
		webhookRepo := &mocks.MockWebhookEventRepo{}

		billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		customer := &domain.Customer{ID: testCustomerID, Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
		purchaseRepo := &mocks.MockPurchaseRepo{}
		creditRepo := &mocks.MockReplayCreditRepo{}
		subRepo := &mocks.MockSubscriptionRepo{}
		customerRepo := &mocks.MockCustomerRepo{}
		webhookRepo := &mocks.MockWebhookEventRepo{}

		billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		customer := &domain.Customer{ID: testCustomerID, Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
			&mocks.MockPurchaseRepo{}, &mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{},
			&mocks.MockCustomerRepo{}, &mocks.MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
		r.Post("/billing/checkout", h.CreateCheckout)

		body := map[string]string{"type": "invalid_type"}
		rec := doRequest(r, "POST", "/billing/checkout", body, testJWT(testCustomerID, false))

		assert.Equal(t, 400, rec.Code)
	})

	t.Run("missing type returns 400", func(t *testing.T) {
		billingSvc := nbNewTestBillingService(
			&mocks.MockPurchaseRepo{}, &mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{},
			&mocks.MockCustomerRepo{}, &mocks.MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
			&mocks.MockPurchaseRepo{}, &mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{},
			&mocks.MockCustomerRepo{}, &mocks.MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
		purchaseRepo := &mocks.MockPurchaseRepo{}
		creditRepo := &mocks.MockReplayCreditRepo{}
		subRepo := &mocks.MockSubscriptionRepo{}
		customerRepo := &mocks.MockCustomerRepo{}
		webhookRepo := &mocks.MockWebhookEventRepo{}

		billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		customer := &domain.Customer{ID: testCustomerID, Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
			&mocks.MockPurchaseRepo{}, &mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{},
			&mocks.MockCustomerRepo{}, &mocks.MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
			&mocks.MockPurchaseRepo{}, &mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{},
			&mocks.MockCustomerRepo{}, &mocks.MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
		purchaseRepo := &mocks.MockPurchaseRepo{}
		creditRepo := &mocks.MockReplayCreditRepo{}
		subRepo := &mocks.MockSubscriptionRepo{}
		customerRepo := &mocks.MockCustomerRepo{}
		webhookRepo := &mocks.MockWebhookEventRepo{}

		billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		customer := &domain.Customer{ID: testCustomerID, Email: "test@example.com", Name: "Test"}
		customerRepo.On("GetByID", mock.Anything, testCustomerID).Return(customer, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
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
			&mocks.MockPurchaseRepo{}, &mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{},
			&mocks.MockCustomerRepo{}, &mocks.MockWebhookEventRepo{},
		)
		quotaSvc := nbNewTestQuotaService(
			&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
			&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
		)
		h := NewBillingHandler(billingSvc, quotaSvc, nbBillingConfig(), testLogger())

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testAuth.KeyFunc(), testAuthIssuer, testAuth.Lookup))
		r.Post("/billing/portal", h.CreatePortalSession)

		rec := doRequest(r, "POST", "/billing/portal", nil, "")
		assert.Equal(t, 401, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Webhook tests
// ---------------------------------------------------------------------------
//
// These tests exercise the real signature-verification path: every payload is
// signed with webhook.ComputeSignature (stripe-go's own test helper), so
// BillingHandler.Webhook -> webhook.ConstructEventWithOptions genuinely
// verifies it. No signature logic is stubbed — that would defeat the point.

// nbNewWebhookHandler wires a BillingHandler to the given mock repos (backing a
// real *service.BillingService) with the test webhook secret. The Webhook
// handler only ever reaches billingService -> webhookRepo and the per-event
// business repos; QuotaService is unused by Webhook but required by the
// constructor.
func nbNewWebhookHandler(
	purchaseRepo *mocks.MockPurchaseRepo,
	creditRepo *mocks.MockReplayCreditRepo,
	subRepo *mocks.MockSubscriptionRepo,
	customerRepo *mocks.MockCustomerRepo,
	webhookRepo *mocks.MockWebhookEventRepo,
) *BillingHandler {
	billingSvc := nbNewTestBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo)
	quotaSvc := nbNewTestQuotaService(
		&mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{}, &mocks.MockEventUsageRepo{},
		&mocks.MockPixelRepo{}, &mocks.MockSalePageRepo{}, &mocks.MockCustomerRepo{},
	)
	return NewBillingHandler(billingSvc, quotaSvc, nbBillingConfigWithWebhook(), testLogger())
}

// nbEventPayload marshals a Stripe webhook event envelope. topObject is the
// top-level "object" field ("event" for a genuine event); dataObject becomes
// event.Data.Raw, which the handler unmarshals into the per-type Stripe struct.
func nbEventPayload(topObject, eventID, eventType string, dataObject map[string]interface{}) []byte {
	payload := map[string]interface{}{
		"id":     eventID,
		"object": topObject,
		"type":   eventType,
		"data":   map[string]interface{}{"object": dataObject},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal webhook event payload: " + err.Error())
	}
	return b
}

// nbSignPayload builds the Stripe-Signature header value for payload using
// webhook.ComputeSignature (the SDK's own test helper). The format
// t=<unix>,v1=<hex> matches what parseSignatureHeader in webhook/client.go
// expects.
func nbSignPayload(payload []byte, secret string, ts time.Time) string {
	sig := webhook.ComputeSignature(ts, payload, secret)
	return fmt.Sprintf("t=%d,v1=%s", ts.Unix(), hex.EncodeToString(sig))
}

// nbSignedWebhookPOST builds a POST request carrying the exact signed payload
// bytes plus the Stripe-Signature header, and returns a recorder to capture it.
func nbSignedWebhookPOST(payload []byte, sigHeader string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	return httptest.NewRecorder(), req
}

func TestBillingHandler_Webhook_ValidCheckoutSessionCompleted(t *testing.T) {
	const evtID = "evt_checkout_completed"
	webhookRepo := &mocks.MockWebhookEventRepo{}

	// Present (but un-expectanted) so we can assert the checkout path leaves
	// them untouched.
	purchaseRepo := &mocks.MockPurchaseRepo{}
	creditRepo := &mocks.MockReplayCreditRepo{}

	h := nbNewWebhookHandler(purchaseRepo, creditRepo,
		&mocks.MockSubscriptionRepo{}, &mocks.MockCustomerRepo{}, webhookRepo)

	// Session has no purchase_id metadata, so HandleCheckoutCompleted returns nil
	// immediately (not a replay_single purchase) without touching the tx pool.
	payload := nbEventPayload("event", evtID, "checkout.session.completed",
		map[string]interface{}{"id": "cs_test_1"})

	webhookRepo.On("CreateIfNotExists", mock.Anything, evtID, "checkout.session.completed").
		Return(true, nil)

	rec, req := nbSignedWebhookPOST(payload, nbSignPayload(payload, nbWebhookSecret, time.Now()))
	h.Webhook(rec, req)

	assert.Equal(t, 200, rec.Code)
	webhookRepo.AssertNumberOfCalls(t, "CreateIfNotExists", 1)
	webhookRepo.AssertExpectations(t)
	// Claim kept (process succeeded) -> no rollback delete.
	webhookRepo.AssertNumberOfCalls(t, "Delete", 0)
}

func TestBillingHandler_Webhook_ValidSubscriptionUpdated(t *testing.T) {
	const (
		evtID     = "evt_sub_updated"
		stripeCus = "cus_test_123"
		stripeSub = "sub_test_1"
	)
	webhookRepo := &mocks.MockWebhookEventRepo{}
	customerRepo := &mocks.MockCustomerRepo{}
	subRepo := &mocks.MockSubscriptionRepo{}
	// Present so we can assert the subscription path did NOT route into checkout.
	purchaseRepo := &mocks.MockPurchaseRepo{}

	h := nbNewWebhookHandler(purchaseRepo, &mocks.MockReplayCreditRepo{}, subRepo, customerRepo, webhookRepo)

	// Empty items -> addonType stays "unknown", so no plan/credit activation runs
	// and the test stays focused on routing.
	payload := nbEventPayload("event", evtID, "customer.subscription.updated",
		map[string]interface{}{
			"id":       stripeSub,
			"customer": stripeCus,
			"status":   "active",
			"items":    map[string]interface{}{"object": "list", "data": []interface{}{}},
		})

	webhookRepo.On("CreateIfNotExists", mock.Anything, evtID, "customer.subscription.updated").
		Return(true, nil)
	customerRepo.On("GetByStripeCustomerID", mock.Anything, stripeCus).
		Return(&domain.Customer{ID: "cust-internal-1"}, nil)
	subRepo.On("GetByStripeSubscriptionID", mock.Anything, stripeSub).Return((*domain.Subscription)(nil), nil)
	subRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Subscription")).Return(nil)

	rec, req := nbSignedWebhookPOST(payload, nbSignPayload(payload, nbWebhookSecret, time.Now()))
	h.Webhook(rec, req)

	assert.Equal(t, 200, rec.Code)

	// Routed down the subscription branch, NOT the checkout branch.
	webhookRepo.AssertCalled(t, "CreateIfNotExists", mock.Anything, evtID, "customer.subscription.updated")
	customerRepo.AssertCalled(t, "GetByStripeCustomerID", mock.Anything, stripeCus)
	subRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*domain.Subscription"))
	purchaseRepo.AssertNumberOfCalls(t, "GetByID", 0)

	webhookRepo.AssertExpectations(t)
	customerRepo.AssertExpectations(t)
	subRepo.AssertExpectations(t)
}

func TestBillingHandler_Webhook_InvalidSignature(t *testing.T) {
	webhookRepo := &mocks.MockWebhookEventRepo{}

	h := nbNewWebhookHandler(
		&mocks.MockPurchaseRepo{}, &mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{},
		&mocks.MockCustomerRepo{}, webhookRepo)

	payload := nbEventPayload("event", "evt_bad_sig", "checkout.session.completed",
		map[string]interface{}{"id": "cs_test_1"})

	// Signed with the WRONG secret — ConstructEvent must reject before any repo
	// is touched.
	rec, req := nbSignedWebhookPOST(payload, nbSignPayload(payload, "whsec_wrong_secret", time.Now()))
	h.Webhook(rec, req)

	assert.Equal(t, 400, rec.Code)
	webhookRepo.AssertNumberOfCalls(t, "CreateIfNotExists", 0)
}

func TestBillingHandler_Webhook_NonEventObjectRejected(t *testing.T) {
	webhookRepo := &mocks.MockWebhookEventRepo{}

	h := nbNewWebhookHandler(
		&mocks.MockPurchaseRepo{}, &mocks.MockReplayCreditRepo{}, &mocks.MockSubscriptionRepo{},
		&mocks.MockCustomerRepo{}, webhookRepo)

	// Correctly signed, but the top-level object is "payment_intent", not "event".
	// This trips checkEventNotification inside ConstructEventWithOptions — the
	// v85 thin-event-notification guard. Regression test: a thin event must NOT
	// be silently accepted as a full Event.
	payload := nbEventPayload("payment_intent", "evt_thin", "payment_intent.created",
		map[string]interface{}{"id": "pi_test_1"})

	rec, req := nbSignedWebhookPOST(payload, nbSignPayload(payload, nbWebhookSecret, time.Now()))
	h.Webhook(rec, req)

	assert.Equal(t, 400, rec.Code)
	webhookRepo.AssertNumberOfCalls(t, "CreateIfNotExists", 0)
}

func TestBillingHandler_Webhook_DuplicateEventIsNoOp(t *testing.T) {
	const evtID = "evt_duplicate"
	webhookRepo := &mocks.MockWebhookEventRepo{}
	purchaseRepo := &mocks.MockPurchaseRepo{}
	creditRepo := &mocks.MockReplayCreditRepo{}

	h := nbNewWebhookHandler(purchaseRepo, creditRepo,
		&mocks.MockSubscriptionRepo{}, &mocks.MockCustomerRepo{}, webhookRepo)

	payload := nbEventPayload("event", evtID, "checkout.session.completed",
		map[string]interface{}{"id": "cs_test_1"})

	// CreateIfNotExists reports the event was already claimed ->
	// ProcessWebhookEvent short-circuits and never invokes the business handler.
	webhookRepo.On("CreateIfNotExists", mock.Anything, evtID, "checkout.session.completed").
		Return(false, nil)

	rec, req := nbSignedWebhookPOST(payload, nbSignPayload(payload, nbWebhookSecret, time.Now()))
	h.Webhook(rec, req)

	assert.Equal(t, 200, rec.Code)
	webhookRepo.AssertNumberOfCalls(t, "CreateIfNotExists", 1)
	webhookRepo.AssertNumberOfCalls(t, "Delete", 0)
	// No purchase / credit write happened downstream.
	purchaseRepo.AssertNumberOfCalls(t, "GetByID", 0)
	purchaseRepo.AssertNumberOfCalls(t, "UpdateStatus", 0)
	creditRepo.AssertNumberOfCalls(t, "Create", 0)

	webhookRepo.AssertExpectations(t)
}

func TestBillingHandler_Webhook_UnknownEventType(t *testing.T) {
	webhookRepo := &mocks.MockWebhookEventRepo{}
	purchaseRepo := &mocks.MockPurchaseRepo{}

	h := nbNewWebhookHandler(purchaseRepo, &mocks.MockReplayCreditRepo{},
		&mocks.MockSubscriptionRepo{}, &mocks.MockCustomerRepo{}, webhookRepo)

	// A type the handler does not switch on -> default branch logs and returns 200.
	payload := nbEventPayload("event", "evt_unknown", "invoice.payment_succeeded",
		map[string]interface{}{"id": "in_test_1"})

	rec, req := nbSignedWebhookPOST(payload, nbSignPayload(payload, nbWebhookSecret, time.Now()))
	h.Webhook(rec, req)

	assert.Equal(t, 200, rec.Code)
	// Never reaches ProcessWebhookEvent, so no repo is called.
	webhookRepo.AssertNumberOfCalls(t, "CreateIfNotExists", 0)
	purchaseRepo.AssertNumberOfCalls(t, "GetByID", 0)
}
