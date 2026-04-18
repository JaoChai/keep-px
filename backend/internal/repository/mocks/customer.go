package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockCustomerRepo is a testify mock for repository.CustomerRepo.
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
func (m *MockCustomerRepo) UpdateRetentionDays(ctx context.Context, customerID string, days int) error {
	args := m.Called(ctx, customerID, days)
	return args.Error(0)
}
func (m *MockCustomerRepo) RegenerateAPIKey(ctx context.Context, customerID, newKey string) (*domain.Customer, error) {
	args := m.Called(ctx, customerID, newKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}
