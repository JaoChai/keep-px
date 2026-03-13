package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var (
	ErrQuotaEventsExceeded       = errors.New("monthly event quota exceeded")
	ErrQuotaPixelsExceeded       = errors.New("pixel limit exceeded")
	ErrQuotaSalePagesExceeded    = errors.New("sale page limit exceeded")
	ErrQuotaReplayNotAllowed     = errors.New("no replay credits available")
	ErrQuotaReplayEventsExceeded = errors.New("replay event count exceeds credit limit")
)

type QuotaService struct {
	creditRepo   repository.ReplayCreditRepository
	subRepo      repository.SubscriptionRepository
	usageRepo    repository.EventUsageRepository
	pixelRepo    repository.PixelRepository
	salePageRepo repository.SalePageRepository
	customerRepo repository.CustomerRepository
}

func NewQuotaService(
	creditRepo repository.ReplayCreditRepository,
	subRepo repository.SubscriptionRepository,
	usageRepo repository.EventUsageRepository,
	pixelRepo repository.PixelRepository,
	salePageRepo repository.SalePageRepository,
	customerRepo repository.CustomerRepository,
) *QuotaService {
	return &QuotaService{
		creditRepo:   creditRepo,
		subRepo:      subRepo,
		usageRepo:    usageRepo,
		pixelRepo:    pixelRepo,
		salePageRepo: salePageRepo,
		customerRepo: customerRepo,
	}
}

// GetCustomerQuota resolves the effective quota limits for a customer
// based on their pixel slot subscription quantity.
func (s *QuotaService) GetCustomerQuota(ctx context.Context, customerID string) (*domain.CustomerQuota, error) {
	// 1. Fetch customer to get plan and retention_days
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil, fmt.Errorf("customer not found: %s", customerID)
	}

	// 2. Get pixel slot quantity from active subscriptions
	slots, err := s.subRepo.GetPixelSlotQuantity(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get pixel slot quantity: %w", err)
	}

	// 3. Build quota based on slot count
	quota := &domain.CustomerQuota{
		PixelSlots:         slots,
		Plan:               customer.Plan,
		MaxEventsPerReplay: domain.DefaultMaxEventsPerReplay,
	}

	if slots == 0 {
		// Free tier limits
		quota.MaxPixels = domain.FreeMaxPixels
		quota.MaxSalePages = domain.FreeMaxSalePages
		quota.MaxEventsPerMonth = domain.FreeMaxEventsPerMonth
		quota.RetentionDays = customer.RetentionDays
		if quota.RetentionDays == 0 {
			quota.RetentionDays = domain.FreeRetentionDays
		}
	} else {
		// Paid tier: limits scale with slot count
		quota.MaxPixels = slots
		quota.MaxSalePages = slots
		quota.MaxEventsPerMonth = int64(slots) * domain.PaidEventsPerSlot
		quota.RetentionDays = domain.PaidRetentionDays
	}

	// 4. Check active replay credits
	credits, err := s.creditRepo.GetActiveByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get active credits: %w", err)
	}
	if len(credits) > 0 {
		quota.CanReplay = true
		totalRemaining := 0
		for _, c := range credits {
			remaining := c.RemainingReplays()
			if remaining == -1 {
				totalRemaining = -1
				break
			}
			totalRemaining += remaining
		}
		quota.RemainingReplays = totalRemaining
	}

	// 5. Get current month usage
	usage, err := s.usageRepo.GetCurrentMonth(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get current month usage: %w", err)
	}
	if usage != nil {
		quota.EventsUsedThisMonth = usage.EventCount
	}

	return quota, nil
}

// CheckAndIncrementEventQuota atomically checks and increments the event usage
// counter, preventing race conditions in concurrent ingestion.
func (s *QuotaService) CheckAndIncrementEventQuota(ctx context.Context, customerID string, count int64) error {
	maxEvents, err := s.subRepo.GetMaxEventsPerMonth(ctx, customerID)
	if err != nil {
		return fmt.Errorf("get max events per month: %w", err)
	}

	if err := s.usageRepo.CheckAndIncrement(ctx, customerID, count, maxEvents); err != nil {
		if errors.Is(err, repository.ErrQuotaExceeded) {
			return ErrQuotaEventsExceeded
		}
		return err
	}
	return nil
}

// DecrementEventQuota reduces the event usage counter to refund skipped events.
func (s *QuotaService) DecrementEventQuota(ctx context.Context, customerID string, count int64) error {
	return s.usageRepo.DecrementCount(ctx, customerID, count)
}

// CheckPixelCreationQuota checks if the customer can create another pixel.
func (s *QuotaService) CheckPixelCreationQuota(ctx context.Context, customerID string) error {
	quota, err := s.GetCustomerQuota(ctx, customerID)
	if err != nil {
		return err
	}

	count, err := s.pixelRepo.CountByCustomerID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("count pixels: %w", err)
	}

	if count >= quota.MaxPixels {
		return ErrQuotaPixelsExceeded
	}
	return nil
}

// CheckSalePageCreationQuota checks if the customer can create another sale page.
func (s *QuotaService) CheckSalePageCreationQuota(ctx context.Context, customerID string) error {
	quota, err := s.GetCustomerQuota(ctx, customerID)
	if err != nil {
		return err
	}

	count, err := s.salePageRepo.CountByCustomerID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("count sale pages: %w", err)
	}

	if count >= quota.MaxSalePages {
		return ErrQuotaSalePagesExceeded
	}
	return nil
}

// ConsumeReplayCredit atomically consumes one replay from the customer's active credits.
// Uses SELECT ... FOR UPDATE SKIP LOCKED to prevent TOCTOU race conditions.
// Returns the credit used so the caller can access MaxEventsPerReplay.
func (s *QuotaService) ConsumeReplayCredit(ctx context.Context, customerID string, eventCount int) (*domain.ReplayCredit, error) {
	// Atomically consume a credit that can handle the event count.
	// ConsumeOneCredit uses SELECT ... FOR UPDATE SKIP LOCKED to prevent
	// concurrent requests from consuming the same credit.
	credit, err := s.creditRepo.ConsumeOneCredit(ctx, customerID, eventCount)
	if err != nil {
		return nil, fmt.Errorf("consume credit: %w", err)
	}
	if credit == nil {
		// Atomic consume failed — do read-only query to classify the error
		readCredits, readErr := s.creditRepo.GetActiveByCustomerID(ctx, customerID)
		if readErr != nil || len(readCredits) == 0 {
			return nil, ErrQuotaReplayNotAllowed
		}
		// Credits exist but none matched event count constraint
		return nil, ErrQuotaReplayEventsExceeded
	}

	return credit, nil
}
