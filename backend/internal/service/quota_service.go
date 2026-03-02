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
// Pre-flight validation checks credit availability and event limits before consuming.
// Returns the credit used so the caller can access MaxEventsPerReplay.
func (s *QuotaService) ConsumeReplayCredit(ctx context.Context, customerID string, eventCount int) (*domain.ReplayCredit, error) {
	// Pre-flight: check credits exist and event count is within limits before consuming
	credits, err := s.creditRepo.GetActiveByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("check active credits: %w", err)
	}
	if len(credits) == 0 {
		return nil, ErrQuotaReplayNotAllowed
	}

	// Verify at least one credit can handle the event count
	hasValidCredit := false
	for _, c := range credits {
		if c.RemainingReplays() != 0 && (c.MaxEventsPerReplay == 0 || eventCount <= c.MaxEventsPerReplay) {
			hasValidCredit = true
			break
		}
	}
	if !hasValidCredit {
		// Check if it's an event limit issue vs no remaining replays
		hasRemaining := false
		for _, c := range credits {
			if c.RemainingReplays() != 0 {
				hasRemaining = true
				break
			}
		}
		if !hasRemaining {
			return nil, ErrQuotaReplayNotAllowed
		}
		return nil, ErrQuotaReplayEventsExceeded
	}

	// Now atomically consume
	credit, err := s.creditRepo.ConsumeOneCredit(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("consume credit: %w", err)
	}
	if credit == nil {
		return nil, ErrQuotaReplayNotAllowed
	}

	// Final check on consumed credit's actual MaxEventsPerReplay
	if credit.MaxEventsPerReplay > 0 && eventCount > credit.MaxEventsPerReplay {
		return nil, ErrQuotaReplayEventsExceeded
	}
	return credit, nil
}
