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

// setupSlotMocks sets up mocks for a customer with a given slot count, no credits, and optional usage.
func setupSlotMocks(customerRepo *MockCustomerRepo, subRepo *MockSubscriptionRepo, creditRepo *MockReplayCreditRepo, usageRepo *MockEventUsageRepo, plan string, retentionDays int, slots int, usage *domain.EventUsage) {
	customerRepo.On("GetByID", mock.Anything, mock.Anything).Return(&domain.Customer{
		ID:            "cust-1",
		Plan:          plan,
		RetentionDays: retentionDays,
	}, nil)
	subRepo.On("GetPixelSlotQuantity", mock.Anything, mock.Anything).Return(slots, nil)
	creditRepo.On("GetActiveByCustomerID", mock.Anything, mock.Anything).Return([]*domain.ReplayCredit{}, nil)
	if usage != nil {
		usageRepo.On("GetCurrentMonth", mock.Anything, mock.Anything).Return(usage, nil)
	} else {
		usageRepo.On("GetCurrentMonth", mock.Anything, mock.Anything).Return(nil, nil)
	}
}

func TestQuotaService_GetCustomerQuota(t *testing.T) {
	t.Run("free tier - 0 slots uses sandbox limits", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()
		setupSlotMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanSandbox, 0, 0, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, domain.PlanSandbox, quota.Plan)
		assert.Equal(t, 0, quota.PixelSlots)
		assert.Equal(t, domain.FreeMaxPixels, quota.MaxPixels)
		assert.Equal(t, int64(domain.FreeMaxEventsPerMonth), quota.MaxEventsPerMonth)
		assert.Equal(t, domain.FreeRetentionDays, quota.RetentionDays)
		assert.Equal(t, domain.FreeMaxSalePages, quota.MaxSalePages)
		assert.Equal(t, domain.DefaultMaxEventsPerReplay, quota.MaxEventsPerReplay)
		assert.False(t, quota.CanReplay)
		assert.Equal(t, 0, quota.RemainingReplays)
		assert.Equal(t, int64(0), quota.EventsUsedThisMonth)
	})

	t.Run("free tier with custom retention days", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()
		setupSlotMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanSandbox, 14, 0, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, 14, quota.RetentionDays)
	})

	t.Run("1 slot - paid tier limits", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()
		setupSlotMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanPaid, domain.PaidRetentionDays, 1, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, domain.PlanPaid, quota.Plan)
		assert.Equal(t, 1, quota.PixelSlots)
		assert.Equal(t, 1, quota.MaxPixels)
		assert.Equal(t, 1, quota.MaxSalePages)
		assert.Equal(t, int64(domain.PaidEventsPerSlot), quota.MaxEventsPerMonth)
		assert.Equal(t, domain.PaidRetentionDays, quota.RetentionDays)
	})

	t.Run("5 slots - limits scale with slot count", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()
		setupSlotMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanPaid, domain.PaidRetentionDays, 5, nil)

		quota, err := svc.GetCustomerQuota(context.Background(), "cust-1")

		require.NoError(t, err)
		assert.Equal(t, domain.PlanPaid, quota.Plan)
		assert.Equal(t, 5, quota.PixelSlots)
		assert.Equal(t, 5, quota.MaxPixels)
		assert.Equal(t, 5, quota.MaxSalePages)
		assert.Equal(t, int64(5)*domain.PaidEventsPerSlot, quota.MaxEventsPerMonth)
		assert.Equal(t, domain.PaidRetentionDays, quota.RetentionDays)
	})

	t.Run("with active credits", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, _, _, customerRepo := newTestQuotaService()

		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(&domain.Customer{
			ID:   "cust-1",
			Plan: domain.PlanSandbox,
		}, nil)
		subRepo.On("GetPixelSlotQuantity", mock.Anything, "cust-1").Return(0, nil)
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
		subRepo.On("GetPixelSlotQuantity", mock.Anything, "cust-1").Return(0, nil)
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
		setupSlotMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanSandbox, 0, 0, usage)

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
		subRepo.On("GetPixelSlotQuantity", mock.Anything, "cust-1").Return(0, errors.New("db error"))

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
		setupSlotMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanSandbox, 0, 0, nil)

		pixelRepo.On("CountByCustomerID", mock.Anything, "cust-1").Return(0, nil)

		err := svc.CheckPixelCreationQuota(context.Background(), "cust-1")

		assert.NoError(t, err)
		pixelRepo.AssertExpectations(t)
	})

	t.Run("at limit returns error", func(t *testing.T) {
		svc, creditRepo, subRepo, usageRepo, pixelRepo, _, customerRepo := newTestQuotaService()
		setupSlotMocks(customerRepo, subRepo, creditRepo, usageRepo, domain.PlanSandbox, 0, 0, nil)

		pixelRepo.On("CountByCustomerID", mock.Anything, "cust-1").Return(1, nil) // FreeMaxPixels = 1

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
		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1", mock.AnythingOfType("int")).Return(activeCredit, nil)

		credit, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 500)

		require.NoError(t, err)
		assert.Equal(t, "credit-1", credit.ID)
		assert.Equal(t, domain.DefaultMaxEventsPerReplay, credit.MaxEventsPerReplay)
		creditRepo.AssertExpectations(t)
	})

	t.Run("no credits available", func(t *testing.T) {
		svc, creditRepo, _, _, _, _, _ := newTestQuotaService()

		// ConsumeOneCredit returns nil when no credit is available
		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1", mock.AnythingOfType("int")).Return(nil, nil)
		// Fallback read-only query to classify error
		creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{}, nil)

		_, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 500)

		assert.ErrorIs(t, err, ErrQuotaReplayNotAllowed)
		creditRepo.AssertExpectations(t)
	})

	t.Run("exceeds max events per replay", func(t *testing.T) {
		svc, creditRepo, _, _, _, _, _ := newTestQuotaService()

		// ConsumeOneCredit returns nil when event count exceeds limit
		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1", mock.AnythingOfType("int")).Return(nil, nil)
		// Fallback read-only query finds credits exist but none matched
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
		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1", mock.AnythingOfType("int")).Return(activeCredit, nil)

		credit, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 999999)

		require.NoError(t, err)
		assert.Equal(t, "credit-1", credit.ID)
		creditRepo.AssertExpectations(t)
	})

	t.Run("consume credit repo error", func(t *testing.T) {
		svc, creditRepo, _, _, _, _, _ := newTestQuotaService()

		creditRepo.On("ConsumeOneCredit", mock.Anything, "cust-1", mock.AnythingOfType("int")).Return(nil, errors.New("db error"))

		_, err := svc.ConsumeReplayCredit(context.Background(), "cust-1", 500)

		assert.Error(t, err)
		creditRepo.AssertExpectations(t)
	})
}
