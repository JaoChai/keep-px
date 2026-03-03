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
	*MockCustomerRepo,
) {
	creditRepo := new(MockReplayCreditRepo)
	subRepo := new(MockSubscriptionRepo)
	usageRepo := new(MockEventUsageRepo)
	pixelRepo := new(MockPixelRepo)
	salePageRepo := new(MockSalePageRepo)
	customerRepo := new(MockCustomerRepo)

	svc := NewQuotaService(creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo, customerRepo)
	return svc, creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo, customerRepo
}

// setupPlanMocks sets up mocks for a customer with a specific plan, no add-on subscriptions, no credits, and optional usage.
func setupPlanMocks(customerRepo *MockCustomerRepo, subRepo *MockSubscriptionRepo, creditRepo *MockReplayCreditRepo, usageRepo *MockEventUsageRepo, plan string, usage *domain.EventUsage) {
	customerRepo.On("GetByID", mock.Anything, mock.Anything).Return(&domain.Customer{
		ID:   "cust-1",
		Plan: plan,
	}, nil)
	subRepo.On("GetActiveByCustomerID", mock.Anything, mock.Anything).Return([]*domain.Subscription{}, nil)
	creditRepo.On("GetActiveByCustomerID", mock.Anything, mock.Anything).Return([]*domain.ReplayCredit{}, nil)
	if usage != nil {
		usageRepo.On("GetCurrentMonth", mock.Anything, mock.Anything).Return(usage, nil)
	} else {
		usageRepo.On("GetCurrentMonth", mock.Anything, mock.Anything).Return(nil, nil)
	}
}

func TestQuotaService_GetCustomerQuota(t *testing.T) {
	t.Run("sandbox plan defaults", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()
		setupPlanMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanSandbox, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, domain.PlanSandbox, quota.Plan)
		assert.Equal(t, 2, quota.MaxPixels)
		assert.Equal(t, int64(5_000), quota.MaxEventsPerMonth)
		assert.Equal(t, 7, quota.RetentionDays)
		assert.Equal(t, 1, quota.MaxSalePages)
		assert.Equal(t, domain.DefaultMaxEventsPerReplay, quota.MaxEventsPerReplay)
		assert.False(t, quota.CanReplay)
		assert.Equal(t, 0, quota.RemainingReplays)
		assert.Equal(t, int64(0), quota.EventsUsedThisMonth)
	})

	t.Run("launch plan limits", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()
		setupPlanMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanLaunch, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, domain.PlanLaunch, quota.Plan)
		assert.Equal(t, 10, quota.MaxPixels)
		assert.Equal(t, int64(200_000), quota.MaxEventsPerMonth)
		assert.Equal(t, 60, quota.RetentionDays)
		assert.Equal(t, 5, quota.MaxSalePages)
	})

	t.Run("shield plan limits", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()
		setupPlanMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanShield, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, domain.PlanShield, quota.Plan)
		assert.Equal(t, 25, quota.MaxPixels)
		assert.Equal(t, int64(1_000_000), quota.MaxEventsPerMonth)
		assert.Equal(t, 180, quota.RetentionDays)
		assert.Equal(t, 15, quota.MaxSalePages)
	})

	t.Run("vault plan limits", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()
		setupPlanMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanVault, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, domain.PlanVault, quota.Plan)
		assert.Equal(t, 50, quota.MaxPixels)
		assert.Equal(t, int64(5_000_000), quota.MaxEventsPerMonth)
		assert.Equal(t, 365, quota.RetentionDays)
		assert.Equal(t, 30, quota.MaxSalePages)
	})

	t.Run("unknown plan falls back to sandbox", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()
		setupPlanMocks(customerRepo, subRepo, creditRepo, usageRepo, "unknown_plan", nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, 2, quota.MaxPixels)
		assert.Equal(t, int64(5_000), quota.MaxEventsPerMonth)
	})

	t.Run("launch plan with addon stacking", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()
		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(&domain.Customer{
			ID:   "cust-1",
			Plan: domain.PlanLaunch,
		}, nil)
		subRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.Subscription{
			{AddonType: domain.AddonEvents1M, Status: domain.SubStatusActive},
			{AddonType: domain.AddonPixels10, Status: domain.SubStatusActive},
			{AddonType: domain.AddonSalePages10, Status: domain.SubStatusActive},
			{AddonType: domain.SubTypePlanLaunch, Status: domain.SubStatusActive}, // plan sub should be skipped
		}, nil)
		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{}, nil)
		usageRepo.On("GetCurrentMonth", mock.Anything, "cust-1").Return(nil, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, 10+10, quota.MaxPixels)                                    // launch base + pixels_10
		assert.Equal(t, int64(200_000+1_000_000), quota.MaxEventsPerMonth)         // launch base + events_1m
		assert.Equal(t, 5+10, quota.MaxSalePages)                                  // launch base + sale_pages_10
		assert.Equal(t, 60, quota.RetentionDays)                                   // launch base (no retention addon)
	})

	t.Run("with active credits", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()

		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(&domain.Customer{
			ID:   "cust-1",
			Plan: domain.PlanSandbox,
		}, nil)
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
	})

	t.Run("with unlimited credits", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()

		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(&domain.Customer{
			ID:   "cust-1",
			Plan: domain.PlanSandbox,
		}, nil)
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

	t.Run("with usage this month", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()
		usage := &domain.EventUsage{EventCount: 50000}
		setupPlanMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanSandbox, usage)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, int64(50000), quota.EventsUsedThisMonth)
	})

	t.Run("customer repo error", func(t *testing.T) {
		svc, _, _, _, _, _, customerRepo := newTestQuotaService()

		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(nil, errors.New("db error"))

		_, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		assert.Error(t, err)
	})

	t.Run("subscription repo error", func(t *testing.T) {
		svc, _, subRepo, _, _, _, customerRepo := newTestQuotaService()

		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(&domain.Customer{
			ID:   "cust-1",
			Plan: domain.PlanSandbox,
		}, nil)
		subRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return(nil, errors.New("db error"))

		_, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		assert.Error(t, err)
	})
}

func TestQuotaService_CheckAndIncrementEventQuota(t *testing.T) {
	t.Run("under limit passes", func(t *testing.T) {
		svc, _, subRepo, usageRepo, _, _, _ := newTestQuotaService()
		subRepo.On("GetMaxEventsPerMonth", mock.Anything, "cust-1").Return(int64(5000), nil)
		usageRepo.On("CheckAndIncrement", mock.Anything, "cust-1", int64(10), int64(5000)).Return(nil)

		err := svc.CheckAndIncrementEventQuota(context.Background(), "cust-1", 10)

		assert.NoError(t, err)
	})

	t.Run("quota exceeded returns error", func(t *testing.T) {
		svc, _, subRepo, usageRepo, _, _, _ := newTestQuotaService()
		subRepo.On("GetMaxEventsPerMonth", mock.Anything, "cust-1").Return(int64(5000), nil)
		usageRepo.On("CheckAndIncrement", mock.Anything, "cust-1", int64(1), int64(5000)).Return(repository.ErrQuotaExceeded)

		err := svc.CheckAndIncrementEventQuota(context.Background(), "cust-1", 1)

		assert.ErrorIs(t, err, ErrQuotaEventsExceeded)
	})
}

func TestQuotaService_CheckPixelCreationQuota(t *testing.T) {
	t.Run("under limit passes", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, pixelRepo, _, customerRepo := newTestQuotaService()
		setupPlanMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanSandbox, nil)

		pixelRepo.On("CountByCustomerID", mock.Anything, "cust-1").Return(1, nil)

		err := svc.CheckPixelCreationQuota(context.Background(), "cust-1")

		assert.NoError(t, err)
		pixelRepo.AssertExpectations(t)
	})

	t.Run("at limit returns error", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, pixelRepo, _, customerRepo := newTestQuotaService()
		setupPlanMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanSandbox, nil)

		pixelRepo.On("CountByCustomerID", mock.Anything, "cust-1").Return(2, nil) // sandbox limit = 2

		err := svc.CheckPixelCreationQuota(context.Background(), "cust-1")

		assert.ErrorIs(t, err, ErrQuotaPixelsExceeded)
		pixelRepo.AssertExpectations(t)
	})
}

func TestQuotaService_ConsumeReplayCredit(t *testing.T) {
	t.Run("success - has credit", func(t *testing.T) {
		svc, creditRepo, _, _, _, _, _ := newTestQuotaService()

		activeCredit := &domain.ReplayCredit{
			ID:                 "credit-1",
			TotalReplays:       3,
			UsedReplays:        1,
			MaxEventsPerReplay: domain.DefaultMaxEventsPerReplay,
		}
		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{activeCredit}, nil)
		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1", mock.AnythingOfType("int")).Return(activeCredit, nil)

		credit, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 500)

		require.NoError(t, err)
		assert.Equal(t, "credit-1", credit.ID)
		assert.Equal(t, domain.DefaultMaxEventsPerReplay, credit.MaxEventsPerReplay)
		creditRepo.AssertExpectations(t)
	})

	t.Run("no credits available", func(t *testing.T) {
		svc, creditRepo, _, _, _, _, _ := newTestQuotaService()

		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{}, nil)

		_, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 500)

		assert.ErrorIs(t, err, ErrQuotaReplayNotAllowed)
		creditRepo.AssertExpectations(t)
	})

	t.Run("exceeds max events per replay", func(t *testing.T) {
		svc, creditRepo, _, _, _, _, _ := newTestQuotaService()

		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{{
			ID:                 "credit-1",
			TotalReplays:       3,
			UsedReplays:        0,
			MaxEventsPerReplay: 1000,
		}}, nil)

		_, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 5000)

		assert.ErrorIs(t, err, ErrQuotaReplayEventsExceeded)
		creditRepo.AssertExpectations(t)
	})

	t.Run("unlimited events per replay (0 limit)", func(t *testing.T) {
		svc, creditRepo, _, _, _, _, _ := newTestQuotaService()

		activeCredit := &domain.ReplayCredit{
			ID:                 "credit-1",
			TotalReplays:       3,
			UsedReplays:        0,
			MaxEventsPerReplay: 0,
		}
		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{activeCredit}, nil)
		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1", mock.AnythingOfType("int")).Return(activeCredit, nil)

		credit, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 999999)

		require.NoError(t, err)
		assert.Equal(t, "credit-1", credit.ID)
		creditRepo.AssertExpectations(t)
	})

	t.Run("consume credit repo error", func(t *testing.T) {
		svc, creditRepo, _, _, _, _, _ := newTestQuotaService()

		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{{
			ID:                 "credit-1",
			TotalReplays:       3,
			UsedReplays:        0,
			MaxEventsPerReplay: 0,
		}}, nil)
		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1", mock.AnythingOfType("int")).Return(nil, errors.New("db error"))

		_, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 500)

		assert.Error(t, err)
		creditRepo.AssertExpectations(t)
	})
}
