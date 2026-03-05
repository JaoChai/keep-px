package service

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockPurchaseRepo
type MockPurchaseRepo struct{ mock.Mock }

func (m *MockPurchaseRepo) Create(ctx context.Context, purchase *domain.Purchase) error {
	args := m.Called(ctx, purchase)
	return args.Error(0)
}
func (m *MockPurchaseRepo) GetByID(ctx context.Context, id string) (*domain.Purchase, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Purchase), args.Error(1)
}
func (m *MockPurchaseRepo) GetByStripeCheckoutSessionID(ctx context.Context, sessionID string) (*domain.Purchase, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Purchase), args.Error(1)
}
func (m *MockPurchaseRepo) UpdateStatus(ctx context.Context, id string, status string, completedAt *time.Time) error {
	args := m.Called(ctx, id, status, completedAt)
	return args.Error(0)
}
func (m *MockPurchaseRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Purchase, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Purchase), args.Error(1)
}

// MockReplayCreditRepo
type MockReplayCreditRepo struct{ mock.Mock }

func (m *MockReplayCreditRepo) Create(ctx context.Context, credit *domain.ReplayCredit) error {
	args := m.Called(ctx, credit)
	return args.Error(0)
}
func (m *MockReplayCreditRepo) GetByID(ctx context.Context, id string) (*domain.ReplayCredit, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ReplayCredit), args.Error(1)
}
func (m *MockReplayCreditRepo) GetActiveByCustomerID(ctx context.Context, customerID string) ([]*domain.ReplayCredit, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ReplayCredit), args.Error(1)
}
func (m *MockReplayCreditRepo) IncrementUsed(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockReplayCreditRepo) ConsumeOneCredit(ctx context.Context, customerID string, maxEventCount int) (*domain.ReplayCredit, error) {
	args := m.Called(ctx, customerID, maxEventCount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ReplayCredit), args.Error(1)
}

// MockSubscriptionRepo
type MockSubscriptionRepo struct{ mock.Mock }

func (m *MockSubscriptionRepo) Create(ctx context.Context, sub *domain.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}
func (m *MockSubscriptionRepo) GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*domain.Subscription, error) {
	args := m.Called(ctx, stripeSubID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Subscription), args.Error(1)
}
func (m *MockSubscriptionRepo) GetActiveByCustomerID(ctx context.Context, customerID string) ([]*domain.Subscription, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Subscription), args.Error(1)
}
func (m *MockSubscriptionRepo) GetMaxEventsPerMonth(ctx context.Context, customerID string) (int64, error) {
	args := m.Called(ctx, customerID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockSubscriptionRepo) Update(ctx context.Context, sub *domain.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}
func (m *MockSubscriptionRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Subscription, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Subscription), args.Error(1)
}

// MockEventUsageRepo
type MockEventUsageRepo struct{ mock.Mock }

func (m *MockEventUsageRepo) IncrementCount(ctx context.Context, customerID string, count int64) error {
	args := m.Called(ctx, customerID, count)
	return args.Error(0)
}
func (m *MockEventUsageRepo) DecrementCount(ctx context.Context, customerID string, count int64) error {
	args := m.Called(ctx, customerID, count)
	return args.Error(0)
}
func (m *MockEventUsageRepo) GetCurrentMonth(ctx context.Context, customerID string) (*domain.EventUsage, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.EventUsage), args.Error(1)
}
func (m *MockEventUsageRepo) CheckAndIncrement(ctx context.Context, customerID string, count int64, maxAllowed int64) error {
	args := m.Called(ctx, customerID, count, maxAllowed)
	return args.Error(0)
}

// MockSalePageRepo
type MockSalePageRepo struct{ mock.Mock }

func (m *MockSalePageRepo) Create(ctx context.Context, page *domain.SalePage) error {
	args := m.Called(ctx, page)
	return args.Error(0)
}
func (m *MockSalePageRepo) GetByID(ctx context.Context, id string) (*domain.SalePage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SalePage), args.Error(1)
}
func (m *MockSalePageRepo) GetBySlug(ctx context.Context, slug string) (*domain.SalePage, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SalePage), args.Error(1)
}
func (m *MockSalePageRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.SalePage, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.SalePage), args.Error(1)
}
func (m *MockSalePageRepo) CountByCustomerID(ctx context.Context, customerID string) (int, error) {
	args := m.Called(ctx, customerID)
	return args.Int(0), args.Error(1)
}
func (m *MockSalePageRepo) Update(ctx context.Context, page *domain.SalePage) error {
	args := m.Called(ctx, page)
	return args.Error(0)
}
func (m *MockSalePageRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockSalePageRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	args := m.Called(ctx, slug)
	return args.Bool(0), args.Error(1)
}

// MockWebhookEventRepo
type MockWebhookEventRepo struct{ mock.Mock }

func (m *MockWebhookEventRepo) CreateIfNotExists(ctx context.Context, stripeEventID, eventType string) (bool, error) {
	args := m.Called(ctx, stripeEventID, eventType)
	return args.Bool(0), args.Error(1)
}
func (m *MockWebhookEventRepo) Delete(ctx context.Context, stripeEventID string) error {
	args := m.Called(ctx, stripeEventID)
	return args.Error(0)
}

// MockCustomerRepo
type MockCustomerRepo struct{ mock.Mock }

func (m *MockCustomerRepo) Create(ctx context.Context, c *domain.Customer) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}
func (m *MockCustomerRepo) GetByID(ctx context.Context, id string) (*domain.Customer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}
func (m *MockCustomerRepo) GetByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}
func (m *MockCustomerRepo) GetByGoogleID(ctx context.Context, googleID string) (*domain.Customer, error) {
	args := m.Called(ctx, googleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}
func (m *MockCustomerRepo) GetByAPIKey(ctx context.Context, key string) (*domain.Customer, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}
func (m *MockCustomerRepo) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*domain.Customer, error) {
	args := m.Called(ctx, stripeCustomerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}
func (m *MockCustomerRepo) Update(ctx context.Context, c *domain.Customer) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}
func (m *MockCustomerRepo) UpdateStripeCustomerID(ctx context.Context, customerID string, stripeCustomerID string) error {
	args := m.Called(ctx, customerID, stripeCustomerID)
	return args.Error(0)
}
func (m *MockCustomerRepo) UpdatePlan(ctx context.Context, customerID string, plan string) error {
	args := m.Called(ctx, customerID, plan)
	return args.Error(0)
}
func (m *MockCustomerRepo) RegenerateAPIKey(ctx context.Context, customerID, newKey string) (*domain.Customer, error) {
	args := m.Called(ctx, customerID, newKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}

// MockRefreshTokenRepo
type MockRefreshTokenRepo struct{ mock.Mock }

func (m *MockRefreshTokenRepo) Create(ctx context.Context, customerID, tokenHash string, expiresAt time.Time) error {
	args := m.Called(ctx, customerID, tokenHash, expiresAt)
	return args.Error(0)
}
func (m *MockRefreshTokenRepo) GetByTokenHash(ctx context.Context, hash string) (string, time.Time, error) {
	args := m.Called(ctx, hash)
	return args.String(0), args.Get(1).(time.Time), args.Error(2)
}
func (m *MockRefreshTokenRepo) DeleteByCustomerID(ctx context.Context, customerID string) error {
	args := m.Called(ctx, customerID)
	return args.Error(0)
}
func (m *MockRefreshTokenRepo) DeleteByTokenHash(ctx context.Context, hash string) error {
	args := m.Called(ctx, hash)
	return args.Error(0)
}

// MockPixelRepo
type MockPixelRepo struct{ mock.Mock }

func (m *MockPixelRepo) Create(ctx context.Context, p *domain.Pixel) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}
func (m *MockPixelRepo) GetByID(ctx context.Context, id string) (*domain.Pixel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Pixel), args.Error(1)
}
func (m *MockPixelRepo) GetByIDs(ctx context.Context, ids []string) ([]*domain.Pixel, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Pixel), args.Error(1)
}
func (m *MockPixelRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Pixel, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Pixel), args.Error(1)
}
func (m *MockPixelRepo) CountByCustomerID(ctx context.Context, customerID string) (int, error) {
	args := m.Called(ctx, customerID)
	return args.Int(0), args.Error(1)
}
func (m *MockPixelRepo) Update(ctx context.Context, p *domain.Pixel) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}
func (m *MockPixelRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockEventRepo
type MockEventRepo struct{ mock.Mock }

func (m *MockEventRepo) Create(ctx context.Context, e *domain.PixelEvent) (bool, error) {
	args := m.Called(ctx, e)
	return args.Bool(0), args.Error(1)
}
func (m *MockEventRepo) GetByID(ctx context.Context, id string) (*domain.PixelEvent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PixelEvent), args.Error(1)
}
func (m *MockEventRepo) ListByPixelID(ctx context.Context, pixelID string, limit, offset int) ([]*domain.PixelEvent, int, error) {
	args := m.Called(ctx, pixelID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.PixelEvent), args.Int(1), args.Error(2)
}
func (m *MockEventRepo) ListByCustomerID(ctx context.Context, customerID string, pixelID string, limit, offset int) ([]*domain.PixelEvent, int, error) {
	args := m.Called(ctx, customerID, pixelID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.PixelEvent), args.Int(1), args.Error(2)
}
func (m *MockEventRepo) MarkForwarded(ctx context.Context, id string, code int, eventsReceived int) error {
	args := m.Called(ctx, id, code, eventsReceived)
	return args.Error(0)
}
func (m *MockEventRepo) GetEventsForReplay(ctx context.Context, pixelID string, types []string, from, to *time.Time, createdBefore *time.Time) ([]*domain.PixelEvent, error) {
	args := m.Called(ctx, pixelID, types, from, to, createdBefore)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PixelEvent), args.Error(1)
}
func (m *MockEventRepo) CountEventsForReplay(ctx context.Context, pixelID string, types []string, from, to *time.Time) (int, error) {
	args := m.Called(ctx, pixelID, types, from, to)
	return args.Int(0), args.Error(1)
}
func (m *MockEventRepo) GetEventsForReplayPreview(ctx context.Context, pixelID string, types []string, from, to *time.Time, limit int) ([]*domain.PixelEvent, error) {
	args := m.Called(ctx, pixelID, types, from, to, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PixelEvent), args.Error(1)
}
func (m *MockEventRepo) GetDistinctEventTypes(ctx context.Context, pixelID string) ([]string, error) {
	args := m.Called(ctx, pixelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
func (m *MockEventRepo) ListLatestByCustomerID(ctx context.Context, customerID string, pixelID string, limit int) ([]*domain.RealtimeEvent, error) {
	args := m.Called(ctx, customerID, pixelID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.RealtimeEvent), args.Error(1)
}

func (m *MockEventRepo) ListRecentByCustomerID(ctx context.Context, customerID string, since time.Time, pixelID string, limit int) ([]*domain.RealtimeEvent, error) {
	args := m.Called(ctx, customerID, since, pixelID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.RealtimeEvent), args.Error(1)
}
func (m *MockEventRepo) DeleteOlderThan(ctx context.Context, before time.Time, batchSize int) (int64, error) {
	args := m.Called(ctx, before, batchSize)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockEventRepo) DeleteExpiredByPlan(ctx context.Context, batchSize int) (int64, error) {
	args := m.Called(ctx, batchSize)
	return args.Get(0).(int64), args.Error(1)
}

// MockReplaySessionRepo
type MockReplaySessionRepo struct{ mock.Mock }

func (m *MockReplaySessionRepo) Create(ctx context.Context, s *domain.ReplaySession) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}
func (m *MockReplaySessionRepo) GetByID(ctx context.Context, id string) (*domain.ReplaySession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ReplaySession), args.Error(1)
}
func (m *MockReplaySessionRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.ReplaySession, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ReplaySession), args.Error(1)
}
func (m *MockReplaySessionRepo) UpdateProgress(ctx context.Context, id string, replayed, failed int) error {
	args := m.Called(ctx, id, replayed, failed)
	return args.Error(0)
}
func (m *MockReplaySessionRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}
func (m *MockReplaySessionRepo) UpdateStatusWithError(ctx context.Context, id string, status string, errorMsg string) error {
	args := m.Called(ctx, id, status, errorMsg)
	return args.Error(0)
}
func (m *MockReplaySessionRepo) GetStatus(ctx context.Context, id string) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}
func (m *MockReplaySessionRepo) UpdateFailedBatches(ctx context.Context, id string, failedBatchRanges []byte) error {
	args := m.Called(ctx, id, failedBatchRanges)
	return args.Error(0)
}
func (m *MockReplaySessionRepo) CancelSession(ctx context.Context, id string) (*domain.ReplaySession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ReplaySession), args.Error(1)
}
func (m *MockReplaySessionRepo) RecoverOrphanedSessions(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}
func (m *MockReplaySessionRepo) UpdateTotalEvents(ctx context.Context, id string, totalEvents int) error {
	args := m.Called(ctx, id, totalEvents)
	return args.Error(0)
}

// MockNotificationRepo
type MockNotificationRepo struct{ mock.Mock }

func (m *MockNotificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}
func (m *MockNotificationRepo) ListByCustomerID(ctx context.Context, customerID string, limit int) ([]*domain.Notification, error) {
	args := m.Called(ctx, customerID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Notification), args.Error(1)
}
func (m *MockNotificationRepo) CountUnread(ctx context.Context, customerID string) (int, error) {
	args := m.Called(ctx, customerID)
	return args.Int(0), args.Error(1)
}
func (m *MockNotificationRepo) MarkRead(ctx context.Context, id, customerID string) error {
	args := m.Called(ctx, id, customerID)
	return args.Error(0)
}
func (m *MockNotificationRepo) MarkAllRead(ctx context.Context, customerID string) error {
	args := m.Called(ctx, customerID)
	return args.Error(0)
}

// MockAdminRepo
type MockAdminRepo struct{ mock.Mock }

func (m *MockAdminRepo) ListCustomers(ctx context.Context, search, plan, status string, limit, offset int) ([]*domain.Customer, int, error) {
	args := m.Called(ctx, search, plan, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.Customer), args.Int(1), args.Error(2)
}
func (m *MockAdminRepo) GetCustomerDetail(ctx context.Context, id string) (*domain.AdminCustomerDetail, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AdminCustomerDetail), args.Error(1)
}
func (m *MockAdminRepo) SuspendCustomer(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockAdminRepo) ActivateCustomer(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockAdminRepo) GetPlatformStats(ctx context.Context) (*domain.PlatformStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PlatformStats), args.Error(1)
}
func (m *MockAdminRepo) GetRevenueChart(ctx context.Context, days int) ([]*domain.RevenueChartPoint, error) {
	args := m.Called(ctx, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.RevenueChartPoint), args.Error(1)
}
func (m *MockAdminRepo) GetGrowthChart(ctx context.Context, days int) ([]*domain.GrowthChartPoint, error) {
	args := m.Called(ctx, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.GrowthChartPoint), args.Error(1)
}
func (m *MockAdminRepo) ListAllPurchases(ctx context.Context, status string, limit, offset int) ([]*domain.AdminPurchase, int, error) {
	args := m.Called(ctx, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.AdminPurchase), args.Int(1), args.Error(2)
}
func (m *MockAdminRepo) ListAllSubscriptions(ctx context.Context, status string, limit, offset int) ([]*domain.AdminSubscription, int, error) {
	args := m.Called(ctx, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.AdminSubscription), args.Int(1), args.Error(2)
}
func (m *MockAdminRepo) ListCreditGrants(ctx context.Context, limit, offset int) ([]*domain.AdminCreditGrantWithCustomer, int, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.AdminCreditGrantWithCustomer), args.Int(1), args.Error(2)
}
