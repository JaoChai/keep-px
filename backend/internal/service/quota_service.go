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
}

func NewQuotaService(
	creditRepo repository.ReplayCreditRepository,
	subRepo repository.SubscriptionRepository,
	usageRepo repository.EventUsageRepository,
	pixelRepo repository.PixelRepository,
	salePageRepo repository.SalePageRepository,
) *QuotaService {
	return &QuotaService{
		creditRepo:   creditRepo,
		subRepo:      subRepo,
		usageRepo:    usageRepo,
		pixelRepo:    pixelRepo,
		salePageRepo: salePageRepo,
	}
}

// GetCustomerQuota resolves the effective quota limits for a customer.
func (s *QuotaService) GetCustomerQuota(ctx context.Context, customerID string) (*domain.CustomerQuota, error) {
	quota := &domain.CustomerQuota{
		MaxPixels:          domain.FreeMaxPixels,
		MaxEventsPerMonth:  int64(domain.FreeMaxEventsPerMonth),
		RetentionDays:      domain.FreeRetentionDays,
		MaxSalePages:       domain.FreeMaxSalePages,
		MaxEventsPerReplay: domain.FreeMaxEventsPerReplay,
	}

	// Check active subscriptions for add-on upgrades
	subs, err := s.subRepo.GetActiveByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get active subscriptions: %w", err)
	}
	for _, sub := range subs {
		switch sub.AddonType {
		case domain.AddonRetention180:
			if domain.AddonRetention180Days > quota.RetentionDays {
				quota.RetentionDays = domain.AddonRetention180Days
			}
		case domain.AddonRetention365:
			if domain.AddonRetention365Days > quota.RetentionDays {
				quota.RetentionDays = domain.AddonRetention365Days
			}
		case domain.AddonEvents1M:
			quota.MaxEventsPerMonth += int64(domain.Addon1MEventsPerMonth)
		}
	}

	// Check active replay credits
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

	// Get current month usage
	usage, err := s.usageRepo.GetCurrentMonth(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get current month usage: %w", err)
	}
	if usage != nil {
		quota.EventsUsedThisMonth = usage.EventCount
	}

	return quota, nil
}

// CheckEventIngestionQuota checks if the customer can ingest the given number of events.
func (s *QuotaService) CheckEventIngestionQuota(ctx context.Context, customerID string, count int64) error {
	quota, err := s.GetCustomerQuota(ctx, customerID)
	if err != nil {
		return err
	}

	if quota.EventsUsedThisMonth+count > quota.MaxEventsPerMonth {
		return ErrQuotaEventsExceeded
	}
	return nil
}

// CheckPixelCreationQuota checks if the customer can create another pixel.
func (s *QuotaService) CheckPixelCreationQuota(ctx context.Context, customerID string) error {
	quota, err := s.GetCustomerQuota(ctx, customerID)
	if err != nil {
		return err
	}

	pixels, err := s.pixelRepo.ListByCustomerID(ctx, customerID)
	if err != nil {
		return err
	}

	if len(pixels) >= quota.MaxPixels {
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

	pages, err := s.salePageRepo.ListByCustomerID(ctx, customerID)
	if err != nil {
		return err
	}

	if len(pages) >= quota.MaxSalePages {
		return ErrQuotaSalePagesExceeded
	}
	return nil
}

// ConsumeReplayCredit consumes one replay from the customer's active credits.
// Returns the credit used so the caller can access MaxEventsPerReplay.
func (s *QuotaService) ConsumeReplayCredit(ctx context.Context, customerID string, eventCount int) (*domain.ReplayCredit, error) {
	credits, err := s.creditRepo.GetActiveByCustomerID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if len(credits) == 0 {
		return nil, ErrQuotaReplayNotAllowed
	}

	// Use the first active credit (sorted by soonest expiry from repo)
	credit := credits[0]

	if eventCount > credit.MaxEventsPerReplay {
		return nil, ErrQuotaReplayEventsExceeded
	}

	if err := s.creditRepo.IncrementUsed(ctx, credit.ID); err != nil {
		return nil, err
	}

	return credit, nil
}
