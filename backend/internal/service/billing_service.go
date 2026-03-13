package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stripe/stripe-go/v82"
	portalsession "github.com/stripe/stripe-go/v82/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v82/checkout/session"
	stripecustomer "github.com/stripe/stripe-go/v82/customer"
	stripesub "github.com/stripe/stripe-go/v82/subscription"

	"golang.org/x/sync/errgroup"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var (
	ErrInvalidPackType     = errors.New("invalid pack type")
	ErrInvalidCheckoutType = errors.New("invalid checkout type")
	ErrStripeNotConfigured = errors.New("stripe is not configured")
	ErrCustomerNotFound    = errors.New("customer not found")
)

// BillingOverview contains all billing info for a customer.
type BillingOverview struct {
	Plan          string                 `json:"plan"`
	Purchases     []*domain.Purchase     `json:"purchases"`
	Credits       []*domain.ReplayCredit `json:"credits"`
	Subscriptions []*domain.Subscription `json:"subscriptions"`
}

// PackConfig defines a replay pack configuration.
type PackConfig struct {
	PriceID            string
	AmountSatang       int
	TotalReplays       int
	MaxEventsPerReplay int
	ExpiryDays         int
}

type BillingService struct {
	purchaseRepo       repository.PurchaseRepository
	creditRepo         repository.ReplayCreditRepository
	subRepo            repository.SubscriptionRepository
	customerRepo       repository.CustomerRepository
	webhookRepo        repository.WebhookEventRepository
	pool               Pool
	cfg                *config.Config
	replaySingleConfig PackConfig
}

// Pool is the subset of pgxpool.Pool used by BillingService for transactions.
type Pool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func NewBillingService(
	purchaseRepo repository.PurchaseRepository,
	creditRepo repository.ReplayCreditRepository,
	subRepo repository.SubscriptionRepository,
	customerRepo repository.CustomerRepository,
	webhookRepo repository.WebhookEventRepository,
	pool Pool,
	cfg *config.Config,
) *BillingService {
	s := &BillingService{
		purchaseRepo: purchaseRepo,
		creditRepo:   creditRepo,
		subRepo:      subRepo,
		customerRepo: customerRepo,
		webhookRepo:  webhookRepo,
		pool:         pool,
		cfg:          cfg,
	}

	// Set Stripe API key
	if cfg.StripeSecretKey != "" {
		stripe.Key = cfg.StripeSecretKey
	}

	s.replaySingleConfig = PackConfig{
		PriceID:            cfg.StripePriceReplaySingle,
		AmountSatang:       29900,
		TotalReplays:       1,
		MaxEventsPerReplay: domain.DefaultMaxEventsPerReplay,
		ExpiryDays:         90,
	}

	return s
}

// EnsureStripeCustomer creates a Stripe customer if the customer doesn't have one yet.
func (s *BillingService) EnsureStripeCustomer(ctx context.Context, customer *domain.Customer) (string, error) {
	if s.cfg.StripeSecretKey == "" {
		return "", ErrStripeNotConfigured
	}

	if customer.StripeCustomerID != nil && *customer.StripeCustomerID != "" {
		return *customer.StripeCustomerID, nil
	}

	params := &stripe.CustomerParams{
		Email: stripe.String(customer.Email),
		Name:  stripe.String(customer.Name),
		Params: stripe.Params{
			Metadata: map[string]string{
				"customer_id": customer.ID,
			},
		},
	}

	sc, err := stripecustomer.New(params)
	if err != nil {
		return "", fmt.Errorf("create stripe customer: %w", err)
	}

	if err := s.customerRepo.UpdateStripeCustomerID(ctx, customer.ID, sc.ID); err != nil {
		return "", fmt.Errorf("save stripe customer id: %w", err)
	}

	customer.StripeCustomerID = &sc.ID
	return sc.ID, nil
}

// CreatePixelSlotCheckout creates a Stripe Checkout session for a pixel slot subscription.
func (s *BillingService) CreatePixelSlotCheckout(ctx context.Context, customerID string, quantity int) (string, error) {
	if quantity < 1 {
		return "", fmt.Errorf("quantity must be at least 1")
	}

	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return "", fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return "", ErrCustomerNotFound
	}

	stripeCustomerID, err := s.EnsureStripeCustomer(ctx, customer)
	if err != nil {
		return "", err
	}

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(stripeCustomerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(s.cfg.StripePricePixelSlot),
				Quantity: stripe.Int64(int64(quantity)),
			},
		},
		SuccessURL: stripe.String(s.cfg.FrontendURL + "/billing?status=success"),
		CancelURL:  stripe.String(s.cfg.FrontendURL + "/billing?status=cancel"),
		Params: stripe.Params{
			Metadata: map[string]string{
				"customer_id": customerID,
				"type":        domain.SubTypePixelSlots,
				"quantity":    fmt.Sprintf("%d", quantity),
			},
		},
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}

	return sess.URL, nil
}

// CreateReplayCheckout creates a Stripe Checkout session for a replay purchase or subscription.
func (s *BillingService) CreateReplayCheckout(ctx context.Context, customerID string, replayType string) (string, error) {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return "", fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return "", ErrCustomerNotFound
	}

	stripeCustomerID, err := s.EnsureStripeCustomer(ctx, customer)
	if err != nil {
		return "", err
	}

	switch replayType {
	case domain.PackReplaySingle:
		// One-time payment for a single replay credit
		purchase := &domain.Purchase{
			CustomerID:   customerID,
			PackType:     domain.PackReplaySingle,
			AmountSatang: s.replaySingleConfig.AmountSatang,
			Currency:     "THB",
			Status:       domain.PurchaseStatusPending,
		}

		if err := s.purchaseRepo.Create(ctx, purchase); err != nil {
			return "", fmt.Errorf("create purchase: %w", err)
		}

		params := &stripe.CheckoutSessionParams{
			Customer: stripe.String(stripeCustomerID),
			Mode:     stripe.String(string(stripe.CheckoutSessionModePayment)),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					Price:    stripe.String(s.replaySingleConfig.PriceID),
					Quantity: stripe.Int64(1),
				},
			},
			SuccessURL: stripe.String(s.cfg.FrontendURL + "/billing?status=success"),
			CancelURL:  stripe.String(s.cfg.FrontendURL + "/billing?status=cancel"),
			Params: stripe.Params{
				Metadata: map[string]string{
					"purchase_id": purchase.ID,
					"customer_id": customerID,
					"pack_type":   domain.PackReplaySingle,
				},
			},
		}

		sess, err := checkoutsession.New(params)
		if err != nil {
			return "", fmt.Errorf("create checkout session: %w", err)
		}

		return sess.URL, nil

	case domain.SubTypeReplayMonthly:
		// Recurring subscription for monthly replay credits
		params := &stripe.CheckoutSessionParams{
			Customer: stripe.String(stripeCustomerID),
			Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					Price:    stripe.String(s.cfg.StripePriceReplayMonthly),
					Quantity: stripe.Int64(1),
				},
			},
			SuccessURL: stripe.String(s.cfg.FrontendURL + "/billing?status=success"),
			CancelURL:  stripe.String(s.cfg.FrontendURL + "/billing?status=cancel"),
			Params: stripe.Params{
				Metadata: map[string]string{
					"customer_id": customerID,
					"type":        domain.SubTypeReplayMonthly,
				},
			},
		}

		sess, err := checkoutsession.New(params)
		if err != nil {
			return "", fmt.Errorf("create checkout session: %w", err)
		}

		return sess.URL, nil

	default:
		return "", ErrInvalidCheckoutType
	}
}

// UpdatePixelSlotQuantity updates the quantity on an existing pixel slot subscription.
func (s *BillingService) UpdatePixelSlotQuantity(ctx context.Context, customerID string, newQuantity int) (string, error) {
	if newQuantity < 1 {
		return "", fmt.Errorf("quantity must be at least 1")
	}

	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return "", fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return "", ErrCustomerNotFound
	}

	if _, err := s.EnsureStripeCustomer(ctx, customer); err != nil {
		return "", err
	}

	// Find existing pixel_slots subscription
	subs, err := s.subRepo.GetActiveByCustomerID(ctx, customerID)
	if err != nil {
		return "", fmt.Errorf("get active subscriptions: %w", err)
	}

	var existing *domain.Subscription
	for _, sub := range subs {
		if sub.AddonType == domain.SubTypePixelSlots {
			existing = sub
			break
		}
	}

	if existing == nil {
		return "", fmt.Errorf("no active pixel_slots subscription found; use CreatePixelSlotCheckout instead")
	}

	// Use Stripe API to fetch the subscription to get item IDs
	stripeSub, err := stripesub.Get(existing.StripeSubscriptionID, nil)
	if err != nil {
		return "", fmt.Errorf("get stripe subscription: %w", err)
	}

	if len(stripeSub.Items.Data) == 0 {
		return "", fmt.Errorf("stripe subscription has no items")
	}

	// Update the subscription item quantity
	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:       stripe.String(stripeSub.Items.Data[0].ID),
				Quantity: stripe.Int64(int64(newQuantity)),
			},
		},
	}

	if _, err := stripesub.Update(existing.StripeSubscriptionID, params); err != nil {
		return "", fmt.Errorf("update stripe subscription: %w", err)
	}

	// Return portal URL for confirmation
	return s.CreateCustomerPortalSession(ctx, customerID)
}

// CreateCustomerPortalSession creates a Stripe Billing Portal session for subscription management.
func (s *BillingService) CreateCustomerPortalSession(ctx context.Context, customerID string) (string, error) {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return "", fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return "", ErrCustomerNotFound
	}

	stripeCustomerID, err := s.EnsureStripeCustomer(ctx, customer)
	if err != nil {
		return "", err
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(stripeCustomerID),
		ReturnURL: stripe.String(s.cfg.FrontendURL + "/billing"),
	}

	sess, err := portalsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create portal session: %w", err)
	}

	return sess.URL, nil
}

// ProcessWebhookEvent atomically claims the event via CreateIfNotExists, processes it,
// and rolls back the claim on failure so retries can succeed.
func (s *BillingService) ProcessWebhookEvent(ctx context.Context, stripeEventID, eventType string, process func() error) error {
	inserted, err := s.webhookRepo.CreateIfNotExists(ctx, stripeEventID, eventType)
	if err != nil {
		return fmt.Errorf("claim webhook event: %w", err)
	}
	if !inserted {
		slog.Info("skipping duplicate webhook event", "stripe_event_id", stripeEventID, "event_type", eventType)
		return nil
	}

	if err := process(); err != nil {
		// Roll back the claim so Stripe retries can succeed
		if delErr := s.webhookRepo.Delete(ctx, stripeEventID); delErr != nil {
			slog.Error("failed to roll back webhook event claim", "stripe_event_id", stripeEventID, "error", delErr)
		}
		return err
	}

	return nil
}

// HandleCheckoutCompleted processes a completed Stripe Checkout session.
// Only handles replay_single one-time purchases (subscriptions are handled via HandleSubscriptionEvent).
func (s *BillingService) HandleCheckoutCompleted(ctx context.Context, sess *stripe.CheckoutSession) error {
	purchaseID, ok := sess.Metadata["purchase_id"]
	if !ok || purchaseID == "" {
		// Not a replay pack purchase (could be a subscription checkout)
		return nil
	}

	purchase, err := s.purchaseRepo.GetByID(ctx, purchaseID)
	if err != nil {
		return fmt.Errorf("get purchase: %w", err)
	}
	if purchase == nil {
		return fmt.Errorf("purchase not found: %s", purchaseID)
	}

	// Only replay_single uses one-time checkout
	if purchase.PackType != domain.PackReplaySingle {
		return fmt.Errorf("unknown pack type: %s", purchase.PackType)
	}

	packCfg := s.replaySingleConfig
	now := time.Now()

	// Use a transaction if pool is available to ensure atomicity
	if s.pool != nil {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin transaction: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		// Update purchase status within transaction
		_, err = tx.Exec(ctx,
			`UPDATE purchases SET status = $1, completed_at = $2 WHERE id = $3`,
			domain.PurchaseStatusCompleted, &now, purchase.ID,
		)
		if err != nil {
			return fmt.Errorf("update purchase status: %w", err)
		}

		// Create replay credit within transaction
		credit := &domain.ReplayCredit{
			CustomerID:         purchase.CustomerID,
			PurchaseID:         &purchase.ID,
			PackType:           purchase.PackType,
			TotalReplays:       packCfg.TotalReplays,
			UsedReplays:        0,
			MaxEventsPerReplay: packCfg.MaxEventsPerReplay,
			ExpiresAt:          now.AddDate(0, 0, packCfg.ExpiryDays),
		}

		err = tx.QueryRow(ctx,
			`INSERT INTO replay_credits (customer_id, purchase_id, pack_type,
				total_replays, used_replays, max_events_per_replay, expires_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING id, created_at`,
			credit.CustomerID, credit.PurchaseID, credit.PackType,
			credit.TotalReplays, credit.UsedReplays, credit.MaxEventsPerReplay, credit.ExpiresAt,
		).Scan(&credit.ID, &credit.CreatedAt)
		if err != nil {
			return fmt.Errorf("create replay credit: %w", err)
		}

		return tx.Commit(ctx)
	}

	// Fallback: no pool — execute without transaction
	if err := s.purchaseRepo.UpdateStatus(ctx, purchase.ID, domain.PurchaseStatusCompleted, &now); err != nil {
		return fmt.Errorf("update purchase status: %w", err)
	}

	credit := &domain.ReplayCredit{
		CustomerID:         purchase.CustomerID,
		PurchaseID:         &purchase.ID,
		PackType:           purchase.PackType,
		TotalReplays:       packCfg.TotalReplays,
		UsedReplays:        0,
		MaxEventsPerReplay: packCfg.MaxEventsPerReplay,
		ExpiresAt:          now.AddDate(0, 0, packCfg.ExpiryDays),
	}

	if err := s.creditRepo.Create(ctx, credit); err != nil {
		return fmt.Errorf("create replay credit: %w", err)
	}

	return nil
}

// HandleSubscriptionEvent processes Stripe subscription lifecycle events.
func (s *BillingService) HandleSubscriptionEvent(ctx context.Context, sub *stripe.Subscription, eventType string) error {
	switch eventType {
	case "customer.subscription.created", "customer.subscription.updated":
		return s.upsertSubscription(ctx, sub)
	case "customer.subscription.deleted":
		return s.cancelSubscription(ctx, sub)
	default:
		return nil
	}
}

func (s *BillingService) upsertSubscription(ctx context.Context, sub *stripe.Subscription) error {
	// Find customer by Stripe customer ID
	customer, err := s.customerRepo.GetByStripeCustomerID(ctx, sub.Customer.ID)
	if err != nil {
		return fmt.Errorf("get customer by stripe id: %w", err)
	}
	if customer == nil {
		return fmt.Errorf("customer not found for stripe id: %s", sub.Customer.ID)
	}

	// Determine addon type from price and read quantity from Stripe
	var priceID string
	var addonType string
	var quantity int64
	if len(sub.Items.Data) > 0 {
		priceID = sub.Items.Data[0].Price.ID
		addonType = s.resolveAddonType(priceID)
		quantity = sub.Items.Data[0].Quantity
	}

	existing, err := s.subRepo.GetByStripeSubscriptionID(ctx, sub.ID)
	if err != nil {
		return fmt.Errorf("get subscription: %w", err)
	}

	status := string(sub.Status)

	// In stripe-go v82, period dates are on SubscriptionItem, not Subscription.
	var periodStart, periodEnd *time.Time
	if len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		ps := time.Unix(item.CurrentPeriodStart, 0)
		pe := time.Unix(item.CurrentPeriodEnd, 0)
		periodStart = &ps
		periodEnd = &pe
	}

	if existing != nil {
		existing.Status = status
		existing.StripePriceID = priceID
		existing.AddonType = addonType
		existing.Quantity = int(quantity)
		existing.CurrentPeriodStart = periodStart
		existing.CurrentPeriodEnd = periodEnd
		existing.CancelAtPeriodEnd = sub.CancelAtPeriodEnd
		existing.UpdatedAt = time.Now()
		if err := s.subRepo.Update(ctx, existing); err != nil {
			return err
		}
	} else {
		newSub := &domain.Subscription{
			CustomerID:           customer.ID,
			StripeSubscriptionID: sub.ID,
			StripePriceID:        priceID,
			AddonType:            addonType,
			Quantity:             int(quantity),
			Status:               status,
			CurrentPeriodStart:   periodStart,
			CurrentPeriodEnd:     periodEnd,
			CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		}
		if err := s.subRepo.Create(ctx, newSub); err != nil {
			return err
		}
	}

	// Handle pixel_slots activation: upgrade to paid plan
	if addonType == domain.SubTypePixelSlots && status == string(stripe.SubscriptionStatusActive) {
		if err := s.customerRepo.UpdatePlan(ctx, customer.ID, domain.PlanPaid); err != nil {
			return fmt.Errorf("update customer plan to paid: %w", err)
		}
		if err := s.customerRepo.UpdateRetentionDays(ctx, customer.ID, domain.PaidRetentionDays); err != nil {
			return fmt.Errorf("update retention days: %w", err)
		}
	}

	// Handle replay_monthly activation: grant unlimited replay credit for billing period
	if addonType == domain.SubTypeReplayMonthly && status == string(stripe.SubscriptionStatusActive) && periodEnd != nil {
		// Dedup: check if a credit with pack_type=plan_grant and same expiry already exists
		credits, credErr := s.creditRepo.GetActiveByCustomerID(ctx, customer.ID)
		if credErr != nil {
			return fmt.Errorf("check existing credits: %w", credErr)
		}
		alreadyGranted := false
		for _, c := range credits {
			if c.PackType == domain.SubTypeReplayMonthly && c.ExpiresAt.Round(time.Second).Equal(periodEnd.Round(time.Second)) {
				alreadyGranted = true
				break
			}
		}
		if !alreadyGranted {
			credit := &domain.ReplayCredit{
				CustomerID:         customer.ID,
				PackType:           domain.SubTypeReplayMonthly,
				TotalReplays:       -1, // unlimited
				UsedReplays:        0,
				MaxEventsPerReplay: domain.DefaultMaxEventsPerReplay,
				ExpiresAt:          *periodEnd,
			}
			if err := s.creditRepo.Create(ctx, credit); err != nil {
				return fmt.Errorf("grant replay monthly credit: %w", err)
			}
		}
	}

	return nil
}

func (s *BillingService) cancelSubscription(ctx context.Context, sub *stripe.Subscription) error {
	existing, err := s.subRepo.GetByStripeSubscriptionID(ctx, sub.ID)
	if err != nil {
		return fmt.Errorf("get subscription: %w", err)
	}
	if existing == nil {
		return nil
	}

	existing.Status = domain.SubStatusCanceled
	existing.UpdatedAt = time.Now()
	if err := s.subRepo.Update(ctx, existing); err != nil {
		return err
	}

	addonType := existing.AddonType

	if addonType == domain.SubTypePixelSlots {
		// Check if customer has OTHER active pixel_slots subscriptions
		customer, err := s.customerRepo.GetByStripeCustomerID(ctx, sub.Customer.ID)
		if err != nil {
			return fmt.Errorf("get customer for plan revert: %w", err)
		}
		if customer == nil {
			return nil
		}

		subs, err := s.subRepo.GetActiveByCustomerID(ctx, customer.ID)
		if err != nil {
			return fmt.Errorf("get active subscriptions: %w", err)
		}

		hasOtherSlotSubs := false
		for _, s := range subs {
			if s.AddonType == domain.SubTypePixelSlots && s.ID != existing.ID {
				hasOtherSlotSubs = true
				break
			}
		}

		if !hasOtherSlotSubs {
			// Revert to sandbox plan and free retention
			if err := s.customerRepo.UpdatePlan(ctx, customer.ID, domain.PlanSandbox); err != nil {
				return fmt.Errorf("revert customer plan to sandbox: %w", err)
			}
			if err := s.customerRepo.UpdateRetentionDays(ctx, customer.ID, domain.FreeRetentionDays); err != nil {
				return fmt.Errorf("revert retention days: %w", err)
			}
		}
	}

	// replay_monthly: just mark canceled — credits expire naturally at period end

	return nil
}

// resolveAddonType maps a Stripe price ID back to an addon/subscription type.
func (s *BillingService) resolveAddonType(priceID string) string {
	switch priceID {
	case s.cfg.StripePricePixelSlot:
		return domain.SubTypePixelSlots
	case s.cfg.StripePriceReplayMonthly:
		return domain.SubTypeReplayMonthly
	case s.cfg.StripePriceReplaySingle:
		return domain.PackReplaySingle
	default:
		return "unknown"
	}
}

// GetBillingOverview returns all billing information for a customer.
func (s *BillingService) GetBillingOverview(ctx context.Context, customerID string) (*BillingOverview, error) {
	var (
		customer      *domain.Customer
		purchases     []*domain.Purchase
		credits       []*domain.ReplayCredit
		subscriptions []*domain.Subscription
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		customer, err = s.customerRepo.GetByID(gCtx, customerID)
		return err
	})

	g.Go(func() error {
		var err error
		purchases, err = s.purchaseRepo.ListByCustomerID(gCtx, customerID)
		return err
	})

	g.Go(func() error {
		var err error
		credits, err = s.creditRepo.GetActiveByCustomerID(gCtx, customerID)
		return err
	})

	g.Go(func() error {
		var err error
		subscriptions, err = s.subRepo.ListByCustomerID(gCtx, customerID)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get billing overview: %w", err)
	}

	if purchases == nil {
		purchases = []*domain.Purchase{}
	}
	if credits == nil {
		credits = []*domain.ReplayCredit{}
	}
	if subscriptions == nil {
		subscriptions = []*domain.Subscription{}
	}

	plan := domain.PlanSandbox
	if customer != nil {
		plan = customer.Plan
	}

	return &BillingOverview{
		Plan:          plan,
		Purchases:     purchases,
		Credits:       credits,
		Subscriptions: subscriptions,
	}, nil
}
