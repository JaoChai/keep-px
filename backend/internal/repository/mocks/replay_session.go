package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockReplaySessionRepo is a testify mock for repository.ReplaySessionRepo.
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
