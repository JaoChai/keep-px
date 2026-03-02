package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

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
	ErrInvalidPackType     = errors.New("invalid pack type")
	ErrInvalidAddonType    = errors.New("invalid addon type")
	ErrStripeNotConfigured = errors.New("stripe is not configured")
	ErrCustomerNotFound    = errors.New("customer not found")
)

// BillingOverview contains all billing info for a customer.
type BillingOverview struct {
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
	purchaseRepo repository.PurchaseRepository
	creditRepo   repository.ReplayCreditRepository
	subRepo      repository.SubscriptionRepository
	customerRepo repository.CustomerRepository
	webhookRepo  repository.WebhookEventRepository
	cfg          *config.Config
	packConfigs  map[string]PackConfig
}

func NewBillingService(
	purchaseRepo repository.PurchaseRepository,
	creditRepo repository.ReplayCreditRepository,
	subRepo repository.SubscriptionRepository,
	customerRepo repository.CustomerRepository,
	webhookRepo repository.WebhookEventRepository,
	cfg *config.Config,
) *BillingService {
	s := &BillingService{
		purchaseRepo: purchaseRepo,
		creditRepo:   creditRepo,
		subRepo:      subRepo,
		customerRepo: customerRepo,
		webhookRepo:  webhookRepo,
		cfg:          cfg,
	}

	// Set Stripe API key
	if cfg.StripeSecretKey != "" {
		stripe.Key = cfg.StripeSecretKey
	}

	s.packConfigs = map[string]PackConfig{
		domain.PackReplay1: {
			PriceID:            cfg.StripePriceReplay1,
			AmountSatang:       29900,
			TotalReplays:       1,
			MaxEventsPerReplay: domain.FreeMaxEventsPerReplay,
			ExpiryDays:         90,
		},
		domain.PackReplay3: {
			PriceID:            cfg.StripePriceReplay3,
			AmountSatang:       69900,
			TotalReplays:       3,
			MaxEventsPerReplay: domain.FreeMaxEventsPerReplay,
			ExpiryDays:         180,
		},
		domain.PackReplayUnlimited: {
			PriceID:            cfg.StripePriceReplayUnlimited,
			AmountSatang:       149000,
			TotalReplays:       -1,
			MaxEventsPerReplay: domain.FreeMaxEventsPerReplay,
			ExpiryDays:         365,
		},
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

	// Update purchase status
	now := time.Now()
	if err := s.purchaseRepo.UpdateStatus(ctx, purchase.ID, domain.PurchaseStatusCompleted, &now); err != nil {
		return fmt.Errorf("update purchase status: %w", err)
	}

	// Create replay credit
	packCfg, ok := s.packConfigs[purchase.PackType]
	if !ok {
		return fmt.Errorf("unknown pack type: %s", purchase.PackType)
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
		return s.subRepo.Update(ctx, existing)
	}

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
	return s.subRepo.Create(ctx, newSub)
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
	return s.subRepo.Update(ctx, existing)
}

// GetBillingOverview returns all billing information for a customer.
func (s *BillingService) GetBillingOverview(ctx context.Context, customerID string) (*BillingOverview, error) {
	var (
		purchases     []*domain.Purchase
		credits       []*domain.ReplayCredit
		subscriptions []*domain.Subscription
	)

	g, gCtx := errgroup.WithContext(ctx)

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

	return &BillingOverview{
		Purchases:     purchases,
		Credits:       credits,
		Subscriptions: subscriptions,
	}, nil
}

// addonPriceID maps an addon type to its Stripe price ID.
func (s *BillingService) addonPriceID(addonType string) (string, error) {
	switch addonType {
	case domain.AddonRetention180:
		return s.cfg.StripePriceRetention180, nil
	case domain.AddonRetention365:
		return s.cfg.StripePriceRetention365, nil
	case domain.AddonEvents1M:
		return s.cfg.StripePriceEvents1M, nil
	default:
		return "", fmt.Errorf("unknown addon type: %s", addonType)
	}
}

// resolveAddonType maps a Stripe price ID back to an addon type.
func (s *BillingService) resolveAddonType(priceID string) string {
	switch priceID {
	case s.cfg.StripePriceRetention180:
		return domain.AddonRetention180
	case s.cfg.StripePriceRetention365:
		return domain.AddonRetention365
	case s.cfg.StripePriceEvents1M:
		return domain.AddonEvents1M
	default:
		return "unknown"
	}
}
