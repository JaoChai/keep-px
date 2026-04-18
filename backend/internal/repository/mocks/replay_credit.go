package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockReplayCreditRepo is a testify mock for repository.ReplayCreditRepo.
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
func (m *MockReplayCreditRepo) RefundCredit(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
