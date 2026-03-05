package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// ErrNotFound is returned by repository implementations when the requested
// entity does not exist or was not affected by the operation.
var ErrNotFound = errors.New("not found")

// ErrQuotaExceeded is returned when an atomic quota check-and-increment
// determines that the requested operation would exceed the allowed limit.
var ErrQuotaExceeded = errors.New("quota exceeded")

type CustomerRepository interface {
	Create(ctx context.Context, customer *domain.Customer) error
	GetByID(ctx context.Context, id string) (*domain.Customer, error)
	GetByEmail(ctx context.Context, email string) (*domain.Customer, error)
	GetByGoogleID(ctx context.Context, googleID string) (*domain.Customer, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*domain.Customer, error)
	GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*domain.Customer, error)
	Update(ctx context.Context, customer *domain.Customer) error
	UpdatePlan(ctx context.Context, customerID string, plan string) error
	UpdateStripeCustomerID(ctx context.Context, customerID string, stripeCustomerID string) error
	RegenerateAPIKey(ctx context.Context, customerID, newKey string) (*domain.Customer, error)
}

type PixelRepository interface {
	Create(ctx context.Context, pixel *domain.Pixel) error
	GetByID(ctx context.Context, id string) (*domain.Pixel, error)
	GetByIDs(ctx context.Context, ids []string) ([]*domain.Pixel, error)
	ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Pixel, error)
	CountByCustomerID(ctx context.Context, customerID string) (int, error)
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
	DeleteOlderThan(ctx context.Context, before time.Time, batchSize int) (int64, error)
	DeleteExpiredByPlan(ctx context.Context, batchSize int) (int64, error)
}

type ReplaySessionRepository interface {
	Create(ctx context.Context, session *domain.ReplaySession) error
	GetByID(ctx context.Context, id string) (*domain.ReplaySession, error)
	ListByCustomerID(ctx context.Context, customerID string) ([]*domain.ReplaySession, error)
	UpdateProgress(ctx context.Context, id string, replayed, failed int) error
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateStatusWithError(ctx context.Context, id string, status string, errorMsg string) error
	UpdateTotalEvents(ctx context.Context, id string, total int) error
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
	CountByCustomerID(ctx context.Context, customerID string) (int, error)
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

type PurchaseRepository interface {
	Create(ctx context.Context, purchase *domain.Purchase) error
	GetByID(ctx context.Context, id string) (*domain.Purchase, error)
	GetByStripeCheckoutSessionID(ctx context.Context, sessionID string) (*domain.Purchase, error)
	UpdateStatus(ctx context.Context, id string, status string, completedAt *time.Time) error
	ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Purchase, error)
}

type ReplayCreditRepository interface {
	Create(ctx context.Context, credit *domain.ReplayCredit) error
	GetByID(ctx context.Context, id string) (*domain.ReplayCredit, error)
	GetActiveByCustomerID(ctx context.Context, customerID string) ([]*domain.ReplayCredit, error)
	IncrementUsed(ctx context.Context, id string) error
	ConsumeOneCredit(ctx context.Context, customerID string, maxEventCount int) (*domain.ReplayCredit, error)
}

type SubscriptionRepository interface {
	Create(ctx context.Context, sub *domain.Subscription) error
	GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*domain.Subscription, error)
	GetActiveByCustomerID(ctx context.Context, customerID string) ([]*domain.Subscription, error)
	GetMaxEventsPerMonth(ctx context.Context, customerID string) (int64, error)
	Update(ctx context.Context, sub *domain.Subscription) error
	ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Subscription, error)
}

type EventUsageRepository interface {
	IncrementCount(ctx context.Context, customerID string, count int64) error
	DecrementCount(ctx context.Context, customerID string, count int64) error
	GetCurrentMonth(ctx context.Context, customerID string) (*domain.EventUsage, error)
	CheckAndIncrement(ctx context.Context, customerID string, count int64, maxAllowed int64) error
}

type WebhookEventRepository interface {
	CreateIfNotExists(ctx context.Context, stripeEventID string, eventType string) (inserted bool, err error)
	Delete(ctx context.Context, stripeEventID string) error
}

type AdminRepository interface {
	ListCustomers(ctx context.Context, search, plan, status string, limit, offset int) ([]*domain.Customer, int, error)
	GetCustomerDetail(ctx context.Context, id string) (*domain.AdminCustomerDetail, error)
	SuspendCustomer(ctx context.Context, id string) error
	ActivateCustomer(ctx context.Context, id string) error
	GetPlatformStats(ctx context.Context) (*domain.PlatformStats, error)
	GetRevenueChart(ctx context.Context, days int) ([]*domain.RevenueChartPoint, error)
	GetGrowthChart(ctx context.Context, days int) ([]*domain.GrowthChartPoint, error)
	ListAllPurchases(ctx context.Context, status string, limit, offset int) ([]*domain.AdminPurchase, int, error)
	ListAllSubscriptions(ctx context.Context, status string, limit, offset int) ([]*domain.AdminSubscription, int, error)
	ListCreditGrants(ctx context.Context, limit, offset int) ([]*domain.AdminCreditGrantWithCustomer, int, error)

	// F1: Sale Pages
	ListAllSalePages(ctx context.Context, search, customerID string, published *bool, limit, offset int) ([]*domain.AdminSalePage, int, error)
	GetSalePageAdminDetail(ctx context.Context, id string) (*domain.AdminSalePageDetail, error)
	SetSalePagePublished(ctx context.Context, id string, published bool) error
	DeleteSalePageByAdmin(ctx context.Context, id string) error

	// F2: Pixels
	ListAllPixels(ctx context.Context, search, customerID string, active *bool, limit, offset int) ([]*domain.AdminPixel, int, error)
	GetPixelAdminDetail(ctx context.Context, id string) (*domain.AdminPixelDetail, error)
	SetPixelActive(ctx context.Context, id string, active bool) error

	// F3: Replays
	ListAllReplaySessions(ctx context.Context, status, customerID string, limit, offset int) ([]*domain.AdminReplaySession, int, error)
	GetReplaySessionAdminDetail(ctx context.Context, id string) (*domain.AdminReplaySessionDetail, error)

	// F4: Events
	ListAllEvents(ctx context.Context, customerID, pixelID, eventName string, limit, offset int) ([]*domain.AdminEvent, int, error)
	GetEventStats(ctx context.Context, hours int) (*domain.AdminEventStats, error)

	// F5: Audit Log
	CreateAuditLog(ctx context.Context, entry *domain.AuditLogEntry) error
	ListAuditLogs(ctx context.Context, adminID, action, targetCustomerID string, from, to *time.Time, limit, offset int) ([]*domain.AuditLogEntry, int, error)
}
