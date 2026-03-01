package service

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockCustomerRepo
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
func (m *MockCustomerRepo) GetByAPIKey(ctx context.Context, key string) (*domain.Customer, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}
func (m *MockCustomerRepo) Update(ctx context.Context, c *domain.Customer) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

// MockRefreshTokenRepo
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

// MockPixelRepo
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
func (m *MockPixelRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Pixel, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Pixel), args.Error(1)
}
func (m *MockPixelRepo) Update(ctx context.Context, p *domain.Pixel) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}
func (m *MockPixelRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockEventRepo
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
func (m *MockEventRepo) ListByCustomerID(ctx context.Context, customerID string, pixelID string, limit, offset int) ([]*domain.PixelEvent, int, error) {
	args := m.Called(ctx, customerID, pixelID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.PixelEvent), args.Int(1), args.Error(2)
}
func (m *MockEventRepo) MarkForwarded(ctx context.Context, id string, code int, eventsReceived int) error {
	args := m.Called(ctx, id, code, eventsReceived)
	return args.Error(0)
}
func (m *MockEventRepo) GetEventsForReplay(ctx context.Context, pixelID string, types []string, from, to *time.Time) ([]*domain.PixelEvent, error) {
	args := m.Called(ctx, pixelID, types, from, to)
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

// MockReplaySessionRepo
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
