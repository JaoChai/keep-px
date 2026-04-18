package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockSalePageRepo is a testify mock for repository.SalePageRepo.
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
func (m *MockSalePageRepo) ListByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]*domain.SalePage, int, error) {
	args := m.Called(ctx, customerID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.SalePage), args.Int(1), args.Error(2)
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
