package repository

import (
	"context"
	"time"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type CustomerRepository interface {
	Create(ctx context.Context, customer *domain.Customer) error
	GetByID(ctx context.Context, id string) (*domain.Customer, error)
	GetByEmail(ctx context.Context, email string) (*domain.Customer, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*domain.Customer, error)
	Update(ctx context.Context, customer *domain.Customer) error
}

type PixelRepository interface {
	Create(ctx context.Context, pixel *domain.Pixel) error
	GetByID(ctx context.Context, id string) (*domain.Pixel, error)
	ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Pixel, error)
	Update(ctx context.Context, pixel *domain.Pixel) error
	Delete(ctx context.Context, id string) error
}

type EventRepository interface {
	Create(ctx context.Context, event *domain.PixelEvent) (created bool, err error)
	GetByID(ctx context.Context, id string) (*domain.PixelEvent, error)
	ListByPixelID(ctx context.Context, pixelID string, limit, offset int) ([]*domain.PixelEvent, int, error)
	ListByCustomerID(ctx context.Context, customerID string, pixelID string, limit, offset int) ([]*domain.PixelEvent, int, error)
	MarkForwarded(ctx context.Context, id string, responseCode int, eventsReceived int) error
	GetEventsForReplay(ctx context.Context, pixelID string, eventTypes []string, from, to *time.Time, createdBefore *time.Time) ([]*domain.PixelEvent, error)
	CountEventsForReplay(ctx context.Context, pixelID string, eventTypes []string, from, to *time.Time) (int, error)
	GetEventsForReplayPreview(ctx context.Context, pixelID string, eventTypes []string, from, to *time.Time, limit int) ([]*domain.PixelEvent, error)
	GetDistinctEventTypes(ctx context.Context, pixelID string) ([]string, error)
	ListLatestByCustomerID(ctx context.Context, customerID string, pixelID string, limit int) ([]*domain.RealtimeEvent, error)
	ListRecentByCustomerID(ctx context.Context, customerID string, since time.Time, pixelID string, limit int) ([]*domain.RealtimeEvent, error)
}

type ReplaySessionRepository interface {
	Create(ctx context.Context, session *domain.ReplaySession) error
	GetByID(ctx context.Context, id string) (*domain.ReplaySession, error)
	ListByCustomerID(ctx context.Context, customerID string) ([]*domain.ReplaySession, error)
	UpdateProgress(ctx context.Context, id string, replayed, failed int) error
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateStatusWithError(ctx context.Context, id string, status string, errorMsg string) error
	GetStatus(ctx context.Context, id string) (string, error)
	UpdateFailedBatches(ctx context.Context, id string, failedBatchRanges []byte) error
	CancelSession(ctx context.Context, id string) (*domain.ReplaySession, error)
	RecoverOrphanedSessions(ctx context.Context) (int, error)
}

type SalePageRepository interface {
	Create(ctx context.Context, page *domain.SalePage) error
	GetByID(ctx context.Context, id string) (*domain.SalePage, error)
	GetBySlug(ctx context.Context, slug string) (*domain.SalePage, error)
	ListByCustomerID(ctx context.Context, customerID string) ([]*domain.SalePage, error)
	Update(ctx context.Context, page *domain.SalePage) error
	Delete(ctx context.Context, id string) error
	SlugExists(ctx context.Context, slug string) (bool, error)
}

type NotificationRepository interface {
	Create(ctx context.Context, n *domain.Notification) error
	ListByCustomerID(ctx context.Context, customerID string, limit int) ([]*domain.Notification, error)
	CountUnread(ctx context.Context, customerID string) (int, error)
	MarkRead(ctx context.Context, id, customerID string) error
	MarkAllRead(ctx context.Context, customerID string) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, customerID, tokenHash string, expiresAt time.Time) error
	GetByTokenHash(ctx context.Context, tokenHash string) (customerID string, expiresAt time.Time, err error)
	DeleteByCustomerID(ctx context.Context, customerID string) error
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
}
