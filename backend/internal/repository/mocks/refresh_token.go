package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockRefreshTokenRepo is a testify mock for repository.RefreshTokenRepo.
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
