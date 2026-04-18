package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockPurchaseRepo is a testify mock for repository.PurchaseRepo.
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
