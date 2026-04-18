package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockSubscriptionRepo is a testify mock for repository.SubscriptionRepo.
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
func (m *MockSubscriptionRepo) GetPixelSlotQuantity(ctx context.Context, customerID string) (int, error) {
	args := m.Called(ctx, customerID)
	return args.Int(0), args.Error(1)
}
func (m *MockSubscriptionRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Subscription, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Subscription), args.Error(1)
}
