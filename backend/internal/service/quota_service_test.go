package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

func newTestQuotaService() (
	*QuotaService,
	*MockReplayCreditRepo,
	*MockSubscriptionRepo,
	*MockEventUsageRepo,
	*MockPixelRepo,
	*MockSalePageRepo,
) {
	creditRepo := new(MockReplayCreditRepo)
	subRepo := new(MockSubscriptionRepo)
	usageRepo := new(MockEventUsageRepo)
	pixelRepo := new(MockPixelRepo)
	salePageRepo := new(MockSalePageRepo)

	svc := NewQuotaService(creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo)
	return svc, creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo
}

// setupFreePlanMocks sets up mocks for a free plan customer with no credits, no subscriptions, and optional usage.
func setupFreePlanMocks(subRepo *MockSubscriptionRepo, creditRepo *MockReplayCreditRepo, usageRepo *MockEventUsageRepo, usage *domain.EventUsage) {
	subRepo.On("GetActiveByCustomerID", mock.Anything, mock.Anything).Return([]*domain.Subscription{}, nil)
	creditRepo.On("GetActiveByCustomerID", mock.Anything, mock.Anything).Return([]*domain.ReplayCredit{}, nil)
	if usage != nil {
		usageRepo.On("GetCurrentMonth", mock.Anything, mock.Anything).Return(usage, nil)
	} else {
		usageRepo.On("GetCurrentMonth", mock.Anything, mock.Anything).Return(nil, nil)
	}
}

func TestQuotaService_GetCustomerQuota(t *testing.T) {
	t.Run("free plan defaults", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _ := newTestQuotaService()
		setupFreePlanMocks(subRepo, creditRepo, usageRepo, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, domain.FreeMaxPixels, quota.MaxPixels)
		assert.Equal(t, int64(domain.FreeMaxEventsPerMonth), quota.MaxEventsPerMonth)
		assert.Equal(t, domain.FreeRetentionDays, quota.RetentionDays)
		assert.Equal(t, domain.FreeMaxSalePages, quota.MaxSalePages)
		assert.Equal(t, domain.FreeMaxEventsPerReplay, quota.MaxEventsPerReplay)
		assert.False(t, quota.CanReplay)
		assert.Equal(t, 0, quota.RemainingReplays)
		assert.Equal(t, int64(0), quota.EventsUsedThisMonth)

		subRepo.AssertExpectations(t)
		creditRepo.AssertExpectations(t)
		usageRepo.AssertExpectations(t)
	})

	t.Run("with active credits", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _ := newTestQuotaService()

		subRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.Subscription{}, nil)
		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{
			{
				ID:           "credit-1",
				TotalReplays: 3,
				UsedReplays:  1,
				ExpiresAt:    time.Now().Add(24 * time.Hour),
			},
		}, nil)
		usageRepo.On("GetCurrentMonth", mock.Anything, "cust-1").Return(nil, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.True(t, quota.CanReplay)
		assert.Equal(t, 2, quota.RemainingReplays)

		subRepo.AssertExpectations(t)
		creditRepo.AssertExpectations(t)
		usageRepo.AssertExpectations(t)
	})

	t.Run("with unlimited credits", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _ := newTestQuotaService()

		subRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.Subscription{}, nil)
		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{
			{
				ID:           "credit-u",
				TotalReplays: -1,
				UsedReplays:  50,
				ExpiresAt:    time.Now().Add(24 * time.Hour),
			},
		}, nil)
		usageRepo.On("GetCurrentMonth", mock.Anything, "cust-1").Return(nil, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.True(t, quota.CanReplay)
		assert.Equal(t, -1, quota.RemainingReplays)
	})

	t.Run("with subscriptions increase limits", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _ := newTestQuotaService()

		subRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.Subscription{
			{AddonType: domain.AddonRetention365, Status: domain.SubStatusActive},
			{AddonType: domain.AddonEvents1M, Status: domain.SubStatusActive},
		}, nil)
		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{}, nil)
		usageRepo.On("GetCurrentMonth", mock.Anything, "cust-1").Return(nil, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, domain.AddonRetention365Days, quota.RetentionDays)
		assert.Equal(t, int64(domain.FreeMaxEventsPerMonth+domain.Addon1MEventsPerMonth), quota.MaxEventsPerMonth)

		subRepo.AssertExpectations(t)
		creditRepo.AssertExpectations(t)
		usageRepo.AssertExpectations(t)
	})

	t.Run("with usage this month", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _ := newTestQuotaService()
		usage := &domain.EventUsage{EventCount: 50000}
		setupFreePlanMocks(subRepo, creditRepo, usageRepo, usage)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, int64(50000), quota.EventsUsedThisMonth)
	})

	t.Run("subscription repo error", func(t *testing.T) {
		svc, _, subRepo, _, _, _ := newTestQuotaService()

		subRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return(nil, errors.New("db error"))

		_, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		assert.Error(t, err)
		subRepo.AssertExpectations(t)
	})
}

func TestQuotaService_CheckAndIncrementEventQuota(t *testing.T) {
	t.Run("under limit passes", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _ := newTestQuotaService()
		usage := &domain.EventUsage{EventCount: 100000}
		setupFreePlanMocks(subRepo, creditRepo, usageRepo, usage)
		usageRepo.On("CheckAndIncrement", mock.Anything, "cust-1", int64(10), int64(domain.FreeMaxEventsPerMonth)).Return(nil)

		err := svc.CheckAndIncrementEventQuota(context.Background(), "cust-1", 10)

		assert.NoError(t, err)
	})

	t.Run("quota exceeded returns error", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _ := newTestQuotaService()
		usage := &domain.EventUsage{EventCount: int64(domain.FreeMaxEventsPerMonth)}
		setupFreePlanMocks(subRepo, creditRepo, usageRepo, usage)
		usageRepo.On("CheckAndIncrement", mock.Anything, "cust-1", int64(1), int64(domain.FreeMaxEventsPerMonth)).Return(repository.ErrQuotaExceeded)

		err := svc.CheckAndIncrementEventQuota(context.Background(), "cust-1", 1)

		assert.ErrorIs(t, err, ErrQuotaEventsExceeded)
	})
}

func TestQuotaService_CheckPixelCreationQuota(t *testing.T) {
	t.Run("under limit passes", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, pixelRepo, _ := newTestQuotaService()
		setupFreePlanMocks(subRepo, creditRepo, usageRepo, nil)

		pixelRepo.On("ListByCustomerID", mock.Anything, "cust-1").Return([]*domain.Pixel{
			{ID: "p1"}, {ID: "p2"},
		}, nil)

		err := svc.CheckPixelCreationQuota(context.Background(), "cust-1")

		assert.NoError(t, err)
		pixelRepo.AssertExpectations(t)
	})

	t.Run("at limit returns error", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, pixelRepo, _ := newTestQuotaService()
		setupFreePlanMocks(subRepo, creditRepo, usageRepo, nil)

		// Create FreeMaxPixels pixels
		pixels := make([]*domain.Pixel, domain.FreeMaxPixels)
		for i := range pixels {
			pixels[i] = &domain.Pixel{ID: "p" + string(rune('0'+i))}
		}
		pixelRepo.On("ListByCustomerID", mock.Anything, "cust-1").Return(pixels, nil)

		err := svc.CheckPixelCreationQuota(context.Background(), "cust-1")

		assert.ErrorIs(t, err, ErrQuotaPixelsExceeded)
		pixelRepo.AssertExpectations(t)
	})
}

func TestQuotaService_ConsumeReplayCredit(t *testing.T) {
	t.Run("success - has credit", func(t *testing.T) {
		svc, creditRepo, _, _, _, _ := newTestQuotaService()

		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1").Return(&domain.ReplayCredit{
			ID:                 "credit-1",
			TotalReplays:       3,
			UsedReplays:        1,
			MaxEventsPerReplay: domain.FreeMaxEventsPerReplay,
		}, nil)

		credit, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 500)

		require.NoError(t, err)
		assert.Equal(t, "credit-1", credit.ID)
		assert.Equal(t, domain.FreeMaxEventsPerReplay, credit.MaxEventsPerReplay)
		creditRepo.AssertExpectations(t)
	})

	t.Run("no credits available", func(t *testing.T) {
		svc, creditRepo, _, _, _, _ := newTestQuotaService()

		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1").Return(nil, nil)

		_, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 500)

		assert.ErrorIs(t, err, ErrQuotaReplayNotAllowed)
		creditRepo.AssertExpectations(t)
	})

	t.Run("exceeds max events per replay", func(t *testing.T) {
		svc, creditRepo, _, _, _, _ := newTestQuotaService()

		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1").Return(&domain.ReplayCredit{
			ID:                 "credit-1",
			MaxEventsPerReplay: 1000,
		}, nil)

		_, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 5000)

		assert.ErrorIs(t, err, ErrQuotaReplayEventsExceeded)
		creditRepo.AssertExpectations(t)
	})

	t.Run("unlimited events per replay (0 limit)", func(t *testing.T) {
		svc, creditRepo, _, _, _, _ := newTestQuotaService()

		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1").Return(&domain.ReplayCredit{
			ID:                 "credit-1",
			MaxEventsPerReplay: 0,
		}, nil)

		credit, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 999999)

		require.NoError(t, err)
		assert.Equal(t, "credit-1", credit.ID)
		creditRepo.AssertExpectations(t)
	})

	t.Run("consume credit repo error", func(t *testing.T) {
		svc, creditRepo, _, _, _, _ := newTestQuotaService()

		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1").Return(nil, errors.New("db error"))

		_, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 500)

		assert.Error(t, err)
		creditRepo.AssertExpectations(t)
	})
}
