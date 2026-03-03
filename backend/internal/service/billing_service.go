package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stripe/stripe-go/v82"
	portalsession "github.com/stripe/stripe-go/v82/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v82/checkout/session"
	stripecustomer "github.com/stripe/stripe-go/v82/customer"

	"golang.org/x/sync/errgroup"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var (
	ErrInvalidPackType      = errors.New("invalid pack type")
	ErrInvalidAddonType     = errors.New("invalid addon type")
	ErrInvalidPlanType      = errors.New("invalid plan type")
	ErrStripeNotConfigured  = errors.New("stripe is not configured")
	ErrCustomerNotFound     = errors.New("customer not found")
	ErrAlreadyOnPlan        = errors.New("already subscribed to a plan")
	ErrPlanUpgradeRequired  = errors.New("paid plan required for this add-on")
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
	purchaseRepo  repository.PurchaseRepository
	creditRepo    repository.ReplayCreditRepository
	subRepo       repository.SubscriptionRepository
	customerRepo  repository.CustomerRepository
	webhookRepo   repository.WebhookEventRepository
	pool          Pool
	cfg           *config.Config
	packConfigs   map[string]PackConfig
	addonPriceIDs map[string]string // addon type -> Stripe price ID
	planPriceIDs  map[string]string // plan sub type -> Stripe price ID
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

	s.packConfigs = map[string]PackConfig{
		domain.PackReplay1: {
			PriceID:            cfg.StripePriceReplay1,
			AmountSatang:       49900,
			TotalReplays:       1,
			MaxEventsPerReplay: domain.DefaultMaxEventsPerReplay,
			ExpiryDays:         90,
		},
		domain.PackReplay3: {
			PriceID:            cfg.StripePriceReplay3,
			AmountSatang:       99900,
			TotalReplays:       3,
			MaxEventsPerReplay: domain.DefaultMaxEventsPerReplay,
			ExpiryDays:         180,
		},
		domain.PackReplayUnlimited: {
			PriceID:            cfg.StripePriceReplayUnlimited,
			AmountSatang:       299000,
			TotalReplays:       -1,
			MaxEventsPerReplay: domain.DefaultMaxEventsPerReplay,
			ExpiryDays:         365,
		},
	}

	s.addonPriceIDs = map[string]string{
		domain.AddonEvents1M:    cfg.StripePriceEvents1M,
		domain.AddonSalePages10: cfg.StripePriceSalePages10,
		domain.AddonPixels10:    cfg.StripePricePixels10,
	}

	s.planPriceIDs = map[string]string{
		domain.SubTypePlanLaunch: cfg.StripePricePlanLaunch,
		domain.SubTypePlanShield: cfg.StripePricePlanShield,
		domain.SubTypePlanVault:  cfg.StripePricePlanVault,
	}

	return s
}

// isPlanType checks if the addon type is a plan subscription.
func isPlanType(addonType string) bool {
	return strings.HasPrefix(addonType, "plan_")
}

// planNameFromType extracts the plan name from a plan subscription type.
func planNameFromType(subType string) string {
	return strings.TrimPrefix(subType, "plan_")
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

// CreateReplayPackCheckout creates a Stripe Checkout session for a replay pack purchase.
func (s *BillingService) CreateReplayPackCheckout(ctx context.Context, customerID string, packType string) (string, error) {
	packCfg, ok := s.packConfigs[packType]
	if !ok {
		return "", ErrInvalidPackType
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

	// Create pending purchase record
	purchase := &domain.Purchase{
		CustomerID:   customerID,
		PackType:     packType,
		AmountSatang: packCfg.AmountSatang,
		Currency:     "THB",
		Status:       domain.PurchaseStatusPending,
	}

	if err := s.purchaseRepo.Create(ctx, purchase); err != nil {
		return "", fmt.Errorf("create purchase: %w", err)
	}

	// Create Stripe Checkout session
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(stripeCustomerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(packCfg.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.cfg.FrontendURL + "/billing?status=success"),
		CancelURL:  stripe.String(s.cfg.FrontendURL + "/billing?status=cancel"),
		Params: stripe.Params{
			Metadata: map[string]string{
				"purchase_id": purchase.ID,
				"customer_id": customerID,
				"pack_type":   packType,
			},
		},
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}

	return sess.URL, nil
}

// CreateAddonSubscriptionCheckout creates a Stripe Checkout session for an add-on subscription.
func (s *BillingService) CreateAddonSubscriptionCheckout(ctx context.Context, customerID string, addonType string) (string, error) {
	priceID, err := s.addonPriceID(addonType)
	if err != nil {
		return "", ErrInvalidAddonType
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
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.cfg.FrontendURL + "/billing?status=success"),
		CancelURL:  stripe.String(s.cfg.FrontendURL + "/billing?status=cancel"),
		Params: stripe.Params{
			Metadata: map[string]string{
				"customer_id": customerID,
				"addon_type":  addonType,
			},
		},
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}

	return sess.URL, nil
}

// CreatePlanSubscriptionCheckout creates a Stripe Checkout session for a plan subscription.
func (s *BillingService) CreatePlanSubscriptionCheckout(ctx context.Context, customerID string, planType string) (string, error) {
	priceID, ok := s.planPriceIDs[planType]
	if !ok || priceID == "" {
		return "", ErrInvalidPlanType
	}

	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return "", fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return "", ErrCustomerNotFound
	}

	// Check existing plan subscription
	subs, err := s.subRepo.GetActiveByCustomerID(ctx, customerID)
	if err != nil {
		return "", fmt.Errorf("get active subscriptions: %w", err)
	}
	for _, sub := range subs {
		if isPlanType(sub.AddonType) {
			return "", ErrAlreadyOnPlan
		}
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
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.cfg.FrontendURL + "/billing?status=success"),
		CancelURL:  stripe.String(s.cfg.FrontendURL + "/billing?status=cancel"),
		Params: stripe.Params{
			Metadata: map[string]string{
				"customer_id": customerID,
				"plan_type":   planType,
			},
		},
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}

	return sess.URL, nil
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
// It wraps purchase status update + credit creation in a database transaction
// when a pool is available, ensuring atomicity.
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

	packCfg, ok := s.packConfigs[purchase.PackType]
	if !ok {
		return fmt.Errorf("unknown pack type: %s", purchase.PackType)
	}

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

	// Determine addon type from price
	var priceID string
	var addonType string
	if len(sub.Items.Data) > 0 {
		priceID = sub.Items.Data[0].Price.ID
		addonType = s.resolveAddonType(priceID)
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
			Status:               status,
			CurrentPeriodStart:   periodStart,
			CurrentPeriodEnd:     periodEnd,
			CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		}
		if err := s.subRepo.Create(ctx, newSub); err != nil {
			return err
		}
	}

	// If this is a plan subscription, update customer plan and grant credits
	if isPlanType(addonType) && status == string(stripe.SubscriptionStatusActive) {
		planName := planNameFromType(addonType)
		if err := s.customerRepo.UpdatePlan(ctx, customer.ID, planName); err != nil {
			return fmt.Errorf("update customer plan: %w", err)
		}
		if periodEnd != nil {
			if err := s.grantPlanReplayCredits(ctx, customer.ID, planName, *periodEnd); err != nil {
				return fmt.Errorf("grant plan replay credits: %w", err)
			}
		}
	}

	return nil
}

// grantPlanReplayCredits grants replay credits based on PlanLimitsMap.IncludedReplays.
func (s *BillingService) grantPlanReplayCredits(ctx context.Context, customerID string, planName string, periodEnd time.Time) error {
	limits, ok := domain.PlanLimitsMap[planName]
	if !ok || limits.IncludedReplays == 0 {
		return nil // Plan doesn't include replay credits
	}

	// Dedup: check if a credit with pack_type=PackPlanGrant and same expires_at already exists
	credits, err := s.creditRepo.GetActiveByCustomerID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("check existing credits: %w", err)
	}
	for _, c := range credits {
		if c.PackType == domain.PackPlanGrant && c.ExpiresAt.Round(time.Second).Equal(periodEnd.Round(time.Second)) {
			return nil // Already granted for this period
		}
	}

	credit := &domain.ReplayCredit{
		CustomerID:         customerID,
		PackType:           domain.PackPlanGrant,
		TotalReplays:       limits.IncludedReplays,
		UsedReplays:        0,
		MaxEventsPerReplay: domain.DefaultMaxEventsPerReplay,
		ExpiresAt:          periodEnd,
	}

	return s.creditRepo.Create(ctx, credit)
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

	// If this is a plan subscription, revert customer to sandbox
	if isPlanType(existing.AddonType) {
		customer, err := s.customerRepo.GetByStripeCustomerID(ctx, sub.Customer.ID)
		if err != nil {
			return fmt.Errorf("get customer for plan revert: %w", err)
		}
		if customer != nil {
			if err := s.customerRepo.UpdatePlan(ctx, customer.ID, domain.PlanSandbox); err != nil {
				return fmt.Errorf("revert customer plan to sandbox: %w", err)
			}
		}
	}

	return nil
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

// addonPriceID maps an addon type to its Stripe price ID.
func (s *BillingService) addonPriceID(addonType string) (string, error) {
	priceID, ok := s.addonPriceIDs[addonType]
	if !ok {
		return "", fmt.Errorf("unknown addon type: %s", addonType)
	}
	return priceID, nil
}

// resolveAddonType maps a Stripe price ID back to an addon type.
func (s *BillingService) resolveAddonType(priceID string) string {
	for addonType, id := range s.addonPriceIDs {
		if id == priceID {
			return addonType
		}
	}
	for planType, id := range s.planPriceIDs {
		if id == priceID {
			return planType
		}
	}
	return "unknown"
}
