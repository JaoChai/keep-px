package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v82"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

func newTestBillingService() (
	*BillingService,
	*MockPurchaseRepo,
	*MockReplayCreditRepo,
	*MockSubscriptionRepo,
	*MockCustomerRepo,
	*MockWebhookEventRepo,
) {
	purchaseRepo := new(MockPurchaseRepo)
	creditRepo := new(MockReplayCreditRepo)
	subRepo := new(MockSubscriptionRepo)
	customerRepo := new(MockCustomerRepo)
	webhookRepo := new(MockWebhookEventRepo)

	cfg := &config.Config{
		StripeSecretKey:            "", // empty to avoid real Stripe calls
		StripePriceReplay1:         "price_replay_1",
		StripePriceReplay3:         "price_replay_3",
		StripePriceReplayUnlimited: "price_replay_unlimited",
		StripePriceRetention180:    "price_retention_180",
		StripePriceRetention365:    "price_retention_365",
		StripePriceEvents1M:        "price_events_1m",
		StripePriceSalePages25:     "price_sale_pages_25",
		StripePricePixels40:        "price_pixels_40",
		FrontendURL:                "http://localhost:5173",
	}

	svc := NewBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo, cfg)
	return svc, purchaseRepo, creditRepo, subRepo, customerRepo, webhookRepo
}

func TestBillingService_HandleCheckoutCompleted(t *testing.T) {
	tests := []struct {
		name    string
		sess    *stripe.CheckoutSession
		setup   func(*MockPurchaseRepo, *MockReplayCreditRepo)
		wantErr bool
	}{
		{
			name: "success",
			sess: &stripe.CheckoutSession{
				Metadata: map[string]string{
					"purchase_id": "purch-1",
					"customer_id": "cust-1",
					"pack_type":   domain.PackReplay1,
				},
			},
			setup: func(pr *MockPurchaseRepo, cr *MockReplayCreditRepo) {
				pr.On("GetByID", mock.Anything, "purch-1").Return(&domain.Purchase{
					ID:         "purch-1",
					CustomerID: "cust-1",
					PackType:   domain.PackReplay1,
				}, nil)
				pr.On("UpdateStatus", mock.Anything, "purch-1", domain.PurchaseStatusCompleted, mock.AnythingOfType("*time.Time")).Return(nil)
				cr.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplayCredit")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "no purchase_id in metadata (subscription checkout)",
			sess: &stripe.CheckoutSession{
				Metadata: map[string]string{
					"customer_id": "cust-1",
				},
			},
			setup:   func(pr *MockPurchaseRepo, cr *MockReplayCreditRepo) {},
			wantErr: false,
		},
		{
			name: "purchase not found",
			sess: &stripe.CheckoutSession{
				Metadata: map[string]string{
					"purchase_id": "purch-missing",
				},
			},
			setup: func(pr *MockPurchaseRepo, cr *MockReplayCreditRepo) {
				pr.On("GetByID", mock.Anything, "purch-missing").Return(nil, nil)
			},
			wantErr: true,
		},
		{
			name: "purchase repo error",
			sess: &stripe.CheckoutSession{
				Metadata: map[string]string{
					"purchase_id": "purch-err",
				},
			},
			setup: func(pr *MockPurchaseRepo, cr *MockReplayCreditRepo) {
				pr.On("GetByID", mock.Anything, "purch-err").Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, purchaseRepo, creditRepo, _, _, _ := newTestBillingService()
			tc.setup(purchaseRepo, creditRepo)

			err := svc.HandleCheckoutCompleted(context.Background(), tc.sess)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			purchaseRepo.AssertExpectations(t)
			creditRepo.AssertExpectations(t)
		})
	}
}

func TestBillingService_HandleCheckoutCompleted_CreatesCorrectCredit(t *testing.T) {
	svc, purchaseRepo, creditRepo, _, _, _ := newTestBillingService()

	sess := &stripe.CheckoutSession{
		Metadata: map[string]string{
			"purchase_id": "purch-1",
		},
	}

	purchaseRepo.On("GetByID", mock.Anything, "purch-1").Return(&domain.Purchase{
		ID:         "purch-1",
		CustomerID: "cust-1",
		PackType:   domain.PackReplay3,
	}, nil)
	purchaseRepo.On("UpdateStatus", mock.Anything, "purch-1", domain.PurchaseStatusCompleted, mock.AnythingOfType("*time.Time")).Return(nil)

	var capturedCredit *domain.ReplayCredit
	creditRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplayCredit")).
		Run(func(args mock.Arguments) {
			capturedCredit = args.Get(1).(*domain.ReplayCredit)
		}).
		Return(nil)

	err := svc.HandleCheckoutCompleted(context.Background(), sess)
	require.NoError(t, err)

	require.NotNil(t, capturedCredit)
	assert.Equal(t, "cust-1", capturedCredit.CustomerID)
	assert.Equal(t, domain.PackReplay3, capturedCredit.PackType)
	assert.Equal(t, 3, capturedCredit.TotalReplays)
	assert.Equal(t, 0, capturedCredit.UsedReplays)
	assert.Equal(t, domain.FreeMaxEventsPerReplay, capturedCredit.MaxEventsPerReplay)

	purchaseRepo.AssertExpectations(t)
	creditRepo.AssertExpectations(t)
}

func TestBillingService_HandleSubscriptionEvent(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		sub       *stripe.Subscription
		setup     func(*MockSubscriptionRepo, *MockCustomerRepo)
		wantErr   bool
	}{
		{
			name:      "created - new subscription",
			eventType: "customer.subscription.created",
			sub: &stripe.Subscription{
				ID:       "sub_123",
				Customer: &stripe.Customer{ID: "stripe_cust_1"},
				Status:   stripe.SubscriptionStatusActive,
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							Price:              &stripe.Price{ID: "price_retention_180"},
							CurrentPeriodStart: time.Now().Unix(),
							CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour).Unix(),
						},
					},
				},
			},
			setup: func(sr *MockSubscriptionRepo, cr *MockCustomerRepo) {
				cr.On("GetByStripeCustomerID", mock.Anything, "stripe_cust_1").Return(&domain.Customer{
					ID: "cust-1",
				}, nil)
				sr.On("GetByStripeSubscriptionID", mock.Anything, "sub_123").Return(nil, nil)
				sr.On("Create", mock.Anything, mock.AnythingOfType("*domain.Subscription")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "updated - existing subscription",
			eventType: "customer.subscription.updated",
			sub: &stripe.Subscription{
				ID:       "sub_123",
				Customer: &stripe.Customer{ID: "stripe_cust_1"},
				Status:   stripe.SubscriptionStatusActive,
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							Price:              &stripe.Price{ID: "price_retention_365"},
							CurrentPeriodStart: time.Now().Unix(),
							CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour).Unix(),
						},
					},
				},
			},
			setup: func(sr *MockSubscriptionRepo, cr *MockCustomerRepo) {
				cr.On("GetByStripeCustomerID", mock.Anything, "stripe_cust_1").Return(&domain.Customer{
					ID: "cust-1",
				}, nil)
				sr.On("GetByStripeSubscriptionID", mock.Anything, "sub_123").Return(&domain.Subscription{
					ID:                   "local-sub-1",
					CustomerID:           "cust-1",
					StripeSubscriptionID: "sub_123",
				}, nil)
				sr.On("Update", mock.Anything, mock.AnythingOfType("*domain.Subscription")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "deleted - cancels subscription",
			eventType: "customer.subscription.deleted",
			sub: &stripe.Subscription{
				ID:       "sub_456",
				Customer: &stripe.Customer{ID: "stripe_cust_2"},
				Status:   stripe.SubscriptionStatusCanceled,
				Items:    &stripe.SubscriptionItemList{},
			},
			setup: func(sr *MockSubscriptionRepo, cr *MockCustomerRepo) {
				sr.On("GetByStripeSubscriptionID", mock.Anything, "sub_456").Return(&domain.Subscription{
					ID:                   "local-sub-2",
					CustomerID:           "cust-2",
					StripeSubscriptionID: "sub_456",
					Status:               domain.SubStatusActive,
				}, nil)
				sr.On("Update", mock.Anything, mock.MatchedBy(func(s *domain.Subscription) bool {
					return s.Status == domain.SubStatusCanceled
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "deleted - subscription not found",
			eventType: "customer.subscription.deleted",
			sub: &stripe.Subscription{
				ID:       "sub_ghost",
				Customer: &stripe.Customer{ID: "stripe_cust_x"},
				Items:    &stripe.SubscriptionItemList{},
			},
			setup: func(sr *MockSubscriptionRepo, cr *MockCustomerRepo) {
				sr.On("GetByStripeSubscriptionID", mock.Anything, "sub_ghost").Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name:      "unknown event type",
			eventType: "customer.subscription.trial_will_end",
			sub: &stripe.Subscription{
				ID:       "sub_any",
				Customer: &stripe.Customer{ID: "stripe_cust_y"},
				Items:    &stripe.SubscriptionItemList{},
			},
			setup:   func(sr *MockSubscriptionRepo, cr *MockCustomerRepo) {},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _, subRepo, customerRepo, _ := newTestBillingService()
			tc.setup(subRepo, customerRepo)

			err := svc.HandleSubscriptionEvent(context.Background(), tc.sub, tc.eventType)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			subRepo.AssertExpectations(t)
			customerRepo.AssertExpectations(t)
		})
	}
}

func TestBillingService_ProcessWebhookEvent(t *testing.T) {
	t.Run("idempotent - duplicate event skipped", func(t *testing.T) {
		svc, _, _, _, _, webhookRepo := newTestBillingService()

		webhookRepo.On("CreateIfNotExists", mock.Anything, "evt_duplicate", "checkout.session.completed").Return(false, nil)

		processCalled := false
		err := svc.ProcessWebhookEvent(context.Background(), "evt_duplicate", "checkout.session.completed", func() error {
			processCalled = true
			return nil
		})

		assert.NoError(t, err)
		assert.False(t, processCalled, "process function should not be called for duplicate events")
		webhookRepo.AssertExpectations(t)
	})

	t.Run("new event - processes and records", func(t *testing.T) {
		svc, _, _, _, _, webhookRepo := newTestBillingService()

		webhookRepo.On("CreateIfNotExists", mock.Anything, "evt_new", "checkout.session.completed").Return(true, nil)

		processCalled := false
		err := svc.ProcessWebhookEvent(context.Background(), "evt_new", "checkout.session.completed", func() error {
			processCalled = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, processCalled, "process function should be called for new events")
		webhookRepo.AssertExpectations(t)
	})

	t.Run("process error - deletes record for retry", func(t *testing.T) {
		svc, _, _, _, _, webhookRepo := newTestBillingService()

		webhookRepo.On("CreateIfNotExists", mock.Anything, "evt_fail", "checkout.session.completed").Return(true, nil)
		webhookRepo.On("Delete", mock.Anything, "evt_fail").Return(nil)

		processErr := errors.New("processing failed")
		err := svc.ProcessWebhookEvent(context.Background(), "evt_fail", "checkout.session.completed", func() error {
			return processErr
		})

		assert.ErrorIs(t, err, processErr)
		webhookRepo.AssertExpectations(t)
	})

	t.Run("insert error propagated", func(t *testing.T) {
		svc, _, _, _, _, webhookRepo := newTestBillingService()

		webhookRepo.On("CreateIfNotExists", mock.Anything, "evt_db_err", "checkout.session.completed").Return(false, errors.New("db error"))

		err := svc.ProcessWebhookEvent(context.Background(), "evt_db_err", "checkout.session.completed", func() error {
			return nil
		})

		assert.Error(t, err)
		webhookRepo.AssertExpectations(t)
	})
}

func TestBillingService_AddonPriceID(t *testing.T) {
	svc, _, _, _, _, _ := newTestBillingService()

	tests := []struct {
		addonType string
		wantPrice string
		wantErr   bool
	}{
		{domain.AddonRetention180, "price_retention_180", false},
		{domain.AddonRetention365, "price_retention_365", false},
		{domain.AddonEvents1M, "price_events_1m", false},
		{domain.AddonSalePages25, "price_sale_pages_25", false},
		{domain.AddonPixels40, "price_pixels_40", false},
		{"unknown", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.addonType, func(t *testing.T) {
			price, err := svc.addonPriceID(tc.addonType)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantPrice, price)
			}
		})
	}
}

func TestBillingService_ResolveAddonType(t *testing.T) {
	svc, _, _, _, _, _ := newTestBillingService()

	tests := []struct {
		priceID       string
		wantAddonType string
	}{
		{"price_retention_180", domain.AddonRetention180},
		{"price_retention_365", domain.AddonRetention365},
		{"price_events_1m", domain.AddonEvents1M},
		{"price_sale_pages_25", domain.AddonSalePages25},
		{"price_pixels_40", domain.AddonPixels40},
		{"price_unknown", "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.priceID, func(t *testing.T) {
			addonType := svc.resolveAddonType(tc.priceID)
			assert.Equal(t, tc.wantAddonType, addonType)
		})
	}
}

func TestBillingService_CreateReplayPackCheckout(t *testing.T) {
	t.Run("invalid pack type", func(t *testing.T) {
		svc, _, _, _, _, _ := newTestBillingService()

		_, err := svc.CreateReplayPackCheckout(context.Background(), "cust-1", "invalid_pack")

		assert.ErrorIs(t, err, ErrInvalidPackType)
	})

	t.Run("customer not found", func(t *testing.T) {
		svc, _, _, _, customerRepo, _ := newTestBillingService()

		customerRepo.On("GetByID", mock.Anything, "cust-missing").Return(nil, nil)

		_, err := svc.CreateReplayPackCheckout(context.Background(), "cust-missing", domain.PackReplay1)

		assert.ErrorIs(t, err, ErrCustomerNotFound)
		customerRepo.AssertExpectations(t)
	})

	t.Run("stripe not configured", func(t *testing.T) {
		svc, _, _, _, customerRepo, _ := newTestBillingService()

		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(&domain.Customer{
			ID:    "cust-1",
			Email: "test@example.com",
			Name:  "Test",
		}, nil)

		// StripeSecretKey is "" in test config, so EnsureStripeCustomer returns ErrStripeNotConfigured
		_, err := svc.CreateReplayPackCheckout(context.Background(), "cust-1", domain.PackReplay1)

		assert.ErrorIs(t, err, ErrStripeNotConfigured)
		customerRepo.AssertExpectations(t)
	})
}

func TestBillingService_GetBillingOverview(t *testing.T) {
	t.Run("success with data", func(t *testing.T) {
		svc, purchaseRepo, creditRepo, subRepo, _, _ := newTestBillingService()

		purchases := []*domain.Purchase{
			{ID: "p1", CustomerID: "cust-1", PackType: domain.PackReplay1, Status: domain.PurchaseStatusCompleted},
		}
		credits := []*domain.ReplayCredit{
			{ID: "c1", CustomerID: "cust-1", PackType: domain.PackReplay1, TotalReplays: 1},
		}
		subscriptions := []*domain.Subscription{
			{ID: "s1", CustomerID: "cust-1", AddonType: domain.AddonRetention180, Status: domain.SubStatusActive},
		}

		purchaseRepo.On("ListByCustomerID", mock.Anything, "cust-1").Return(purchases, nil)
		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return(credits, nil)
		subRepo.On("ListByCustomerID", mock.Anything, "cust-1").Return(subscriptions, nil)

		overview, err := svc.GetBillingOverview(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Len(t, overview.Purchases, 1)
		assert.Len(t, overview.Credits, 1)
		assert.Len(t, overview.Subscriptions, 1)

		purchaseRepo.AssertExpectations(t)
		creditRepo.AssertExpectations(t)
		subRepo.AssertExpectations(t)
	})

	t.Run("empty data returns empty arrays", func(t *testing.T) {
		svc, purchaseRepo, creditRepo, subRepo, _, _ := newTestBillingService()

		purchaseRepo.On("ListByCustomerID", mock.Anything, "cust-new").Return(nil, nil)
		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-new").Return(nil, nil)
		subRepo.On("ListByCustomerID", mock.Anything, "cust-new").Return(nil, nil)

		overview, err := svc.GetBillingOverview(context.Background(), "cust-new")

		require.NoError(t, err)
		assert.NotNil(t, overview.Purchases)
		assert.NotNil(t, overview.Credits)
		assert.NotNil(t, overview.Subscriptions)
		assert.Empty(t, overview.Purchases)
		assert.Empty(t, overview.Credits)
		assert.Empty(t, overview.Subscriptions)

		purchaseRepo.AssertExpectations(t)
		creditRepo.AssertExpectations(t)
		subRepo.AssertExpectations(t)
	})

	t.Run("purchase repo error", func(t *testing.T) {
		svc, purchaseRepo, creditRepo, subRepo, _, _ := newTestBillingService()

		purchaseRepo.On("ListByCustomerID", mock.Anything, "cust-1").Return(nil, errors.New("db error"))
		// These may or may not be called due to errgroup cancellation
		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return(nil, nil).Maybe()
		subRepo.On("ListByCustomerID", mock.Anything, "cust-1").Return(nil, nil).Maybe()

		_, err := svc.GetBillingOverview(context.Background(), "cust-1")

		assert.Error(t, err)
		purchaseRepo.AssertExpectations(t)
	})
}
