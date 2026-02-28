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
	Create(ctx context.Context, event *domain.PixelEvent) error
	GetByID(ctx context.Context, id string) (*domain.PixelEvent, error)
	ListByPixelID(ctx context.Context, pixelID string, limit, offset int) ([]*domain.PixelEvent, int, error)
	ListByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]*domain.PixelEvent, int, error)
	MarkForwarded(ctx context.Context, id string, responseCode int) error
	GetEventsForReplay(ctx context.Context, pixelID string, eventTypes []string, from, to *time.Time) ([]*domain.PixelEvent, error)
	ListLatestByCustomerID(ctx context.Context, customerID string, pixelID string, limit int) ([]*domain.RealtimeEvent, error)
	ListRecentByCustomerID(ctx context.Context, customerID string, since time.Time, pixelID string, limit int) ([]*domain.RealtimeEvent, error)
}

type EventRuleRepository interface {
	Create(ctx context.Context, rule *domain.EventRule) error
	GetByID(ctx context.Context, id string) (*domain.EventRule, error)
	ListByPixelID(ctx context.Context, pixelID string) ([]*domain.EventRule, error)
	ListActiveByPixelID(ctx context.Context, pixelID string) ([]*domain.EventRule, error)
	Update(ctx context.Context, rule *domain.EventRule) error
	Delete(ctx context.Context, id string) error
}

type ReplaySessionRepository interface {
	Create(ctx context.Context, session *domain.ReplaySession) error
	GetByID(ctx context.Context, id string) (*domain.ReplaySession, error)
	ListByCustomerID(ctx context.Context, customerID string) ([]*domain.ReplaySession, error)
	UpdateProgress(ctx context.Context, id string, replayed, failed int) error
	UpdateStatus(ctx context.Context, id string, status string) error
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

type CustomDomainRepository interface {
	Create(ctx context.Context, d *domain.CustomDomain) error
	GetByID(ctx context.Context, id string) (*domain.CustomDomain, error)
	GetByDomain(ctx context.Context, domainName string) (*domain.CustomDomain, error)
	ListByCustomerID(ctx context.Context, customerID string) ([]*domain.CustomDomain, error)
	ListBySalePageID(ctx context.Context, salePageID string) ([]*domain.CustomDomain, error)
	Update(ctx context.Context, d *domain.CustomDomain) error
	Delete(ctx context.Context, id string) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, customerID, tokenHash string, expiresAt time.Time) error
	GetByTokenHash(ctx context.Context, tokenHash string) (customerID string, expiresAt time.Time, err error)
	DeleteByCustomerID(ctx context.Context, customerID string) error
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
}
