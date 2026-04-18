package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockPixelRepo is a testify mock for repository.PixelRepo.
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
