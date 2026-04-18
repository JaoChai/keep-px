package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockAdminRepo is a testify mock for repository.AdminRepo.
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
func (m *MockAdminRepo) ListAllSalePages(ctx context.Context, search, customerID string, published *bool, limit, offset int) ([]*domain.AdminSalePage, int, error) {
	args := m.Called(ctx, search, customerID, published, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.AdminSalePage), args.Int(1), args.Error(2)
}
func (m *MockAdminRepo) GetSalePageAdminDetail(ctx context.Context, id string) (*domain.AdminSalePageDetail, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AdminSalePageDetail), args.Error(1)
}
func (m *MockAdminRepo) SetSalePagePublished(ctx context.Context, id string, published bool) error {
	args := m.Called(ctx, id, published)
	return args.Error(0)
}
func (m *MockAdminRepo) DeleteSalePageByAdmin(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockAdminRepo) ListAllPixels(ctx context.Context, search, customerID string, active *bool, limit, offset int) ([]*domain.AdminPixel, int, error) {
	args := m.Called(ctx, search, customerID, active, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.AdminPixel), args.Int(1), args.Error(2)
}
func (m *MockAdminRepo) GetPixelAdminDetail(ctx context.Context, id string) (*domain.AdminPixelDetail, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AdminPixelDetail), args.Error(1)
}
func (m *MockAdminRepo) SetPixelActive(ctx context.Context, id string, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}
func (m *MockAdminRepo) ListAllReplaySessions(ctx context.Context, status, customerID string, limit, offset int) ([]*domain.AdminReplaySession, int, error) {
	args := m.Called(ctx, status, customerID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.AdminReplaySession), args.Int(1), args.Error(2)
}
func (m *MockAdminRepo) GetReplaySessionAdminDetail(ctx context.Context, id string) (*domain.AdminReplaySessionDetail, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AdminReplaySessionDetail), args.Error(1)
}
func (m *MockAdminRepo) ListAllEvents(ctx context.Context, customerID, pixelID, eventName string, limit, offset int) ([]*domain.AdminEvent, int, error) {
	args := m.Called(ctx, customerID, pixelID, eventName, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.AdminEvent), args.Int(1), args.Error(2)
}
func (m *MockAdminRepo) GetEventStats(ctx context.Context, hours int) (*domain.AdminEventStats, error) {
	args := m.Called(ctx, hours)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AdminEventStats), args.Error(1)
}
func (m *MockAdminRepo) CreateAuditLog(ctx context.Context, entry *domain.AuditLogEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}
func (m *MockAdminRepo) ListAuditLogs(ctx context.Context, adminID, action, targetCustomerID string, from, to *time.Time, limit, offset int) ([]*domain.AuditLogEntry, int, error) {
	args := m.Called(ctx, adminID, action, targetCustomerID, from, to, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.AuditLogEntry), args.Int(1), args.Error(2)
}
