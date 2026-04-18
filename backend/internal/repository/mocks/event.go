package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockEventRepo is a testify mock for repository.EventRepo.
type MockEventRepo struct{ mock.Mock }

func (m *MockEventRepo) Create(ctx context.Context, e *domain.PixelEvent) (bool, error) {
	args := m.Called(ctx, e)
	return args.Bool(0), args.Error(1)
}
func (m *MockEventRepo) GetByID(ctx context.Context, id string) (*domain.PixelEvent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PixelEvent), args.Error(1)
}
func (m *MockEventRepo) ListByPixelID(ctx context.Context, pixelID string, limit, offset int) ([]*domain.PixelEvent, int, error) {
	args := m.Called(ctx, pixelID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.PixelEvent), args.Int(1), args.Error(2)
}
func (m *MockEventRepo) ListByCustomerID(ctx context.Context, customerID string, pixelID string, eventName string, from, to *time.Time, limit, offset int) ([]*domain.PixelEvent, int, error) {
	args := m.Called(ctx, customerID, pixelID, eventName, from, to, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.PixelEvent), args.Int(1), args.Error(2)
}
func (m *MockEventRepo) MarkForwarded(ctx context.Context, id string, code int, eventsReceived int) error {
	args := m.Called(ctx, id, code, eventsReceived)
	return args.Error(0)
}
func (m *MockEventRepo) GetEventsForReplay(ctx context.Context, pixelID string, types []string, from, to *time.Time, createdBefore *time.Time) ([]*domain.PixelEvent, error) {
	args := m.Called(ctx, pixelID, types, from, to, createdBefore)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PixelEvent), args.Error(1)
}
func (m *MockEventRepo) CountEventsForReplay(ctx context.Context, pixelID string, types []string, from, to *time.Time) (int, error) {
	args := m.Called(ctx, pixelID, types, from, to)
	return args.Int(0), args.Error(1)
}
func (m *MockEventRepo) GetEventsForReplayPreview(ctx context.Context, pixelID string, types []string, from, to *time.Time, limit int) ([]*domain.PixelEvent, error) {
	args := m.Called(ctx, pixelID, types, from, to, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PixelEvent), args.Error(1)
}
func (m *MockEventRepo) GetDistinctEventTypes(ctx context.Context, pixelID string) ([]string, error) {
	args := m.Called(ctx, pixelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
func (m *MockEventRepo) GetDistinctEventTypesByCustomerID(ctx context.Context, customerID string) ([]string, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
func (m *MockEventRepo) ListLatestByCustomerID(ctx context.Context, customerID string, pixelID string, limit int) ([]*domain.RealtimeEvent, error) {
	args := m.Called(ctx, customerID, pixelID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.RealtimeEvent), args.Error(1)
}

func (m *MockEventRepo) ListRecentByCustomerID(ctx context.Context, customerID string, since time.Time, pixelID string, limit int) ([]*domain.RealtimeEvent, error) {
	args := m.Called(ctx, customerID, since, pixelID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.RealtimeEvent), args.Error(1)
}
func (m *MockEventRepo) DeleteOlderThan(ctx context.Context, before time.Time, batchSize int) (int64, error) {
	args := m.Called(ctx, before, batchSize)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockEventRepo) DeleteExpiredByRetention(ctx context.Context, batchSize int) (int64, error) {
	args := m.Called(ctx, batchSize)
	return args.Get(0).(int64), args.Error(1)
}
