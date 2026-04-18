package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockEventUsageRepo is a testify mock for repository.EventUsageRepo.
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
