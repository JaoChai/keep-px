package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

const platformStatsCacheTTL = 5 * time.Minute

// platformStatsCache is a simple TTL cache for GetPlatformOverview results.
type platformStatsCache struct {
	mu        sync.RWMutex
	stats     *domain.PlatformStats
	expiresAt time.Time
}

func (c *platformStatsCache) get() (*domain.PlatformStats, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.stats != nil && time.Now().Before(c.expiresAt) {
		return c.stats, true
	}
	return nil, false
}

func (c *platformStatsCache) set(stats *domain.PlatformStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stats = stats
	c.expiresAt = time.Now().Add(platformStatsCacheTTL)
}

var (
	ErrAdminSelfSuspend = errors.New("cannot suspend your own account")
	ErrInvalidPlan      = errors.New("invalid plan")
)

type AdminService struct {
	adminRepo         repository.AdminRepository
	customerRepo      repository.CustomerRepository
	creditRepo        repository.ReplayCreditRepository
	replaySessionRepo repository.ReplaySessionRepository
	pool              *pgxpool.Pool
	logger            *slog.Logger
	statsCache        platformStatsCache
}

func NewAdminService(
	adminRepo repository.AdminRepository,
	customerRepo repository.CustomerRepository,
	creditRepo repository.ReplayCreditRepository,
	replaySessionRepo repository.ReplaySessionRepository,
	pool *pgxpool.Pool,
	logger *slog.Logger,
) *AdminService {
	return &AdminService{
		adminRepo:         adminRepo,
		customerRepo:      customerRepo,
		creditRepo:        creditRepo,
		replaySessionRepo: replaySessionRepo,
		pool:              pool,
		logger:            logger,
	}
}

func (s *AdminService) logAudit(ctx context.Context, adminID, action, targetType, targetID string, targetCustomerID *string, details any) {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		s.logger.Error("failed to marshal audit details", "action", action, "error", err)
	}
	// Use context.WithoutCancel to prevent client disconnect from aborting audit write
	auditCtx := context.WithoutCancel(ctx)
	if err := s.adminRepo.CreateAuditLog(auditCtx, &domain.AuditLogEntry{
		AdminID:          adminID,
		Action:           action,
		TargetType:       targetType,
		TargetID:         targetID,
		TargetCustomerID: targetCustomerID,
		Details:          detailsJSON,
	}); err != nil {
		s.logger.Error("failed to write audit log", "action", action, "target_type", targetType, "target_id", targetID, "error", err)
	}
}

func (s *AdminService) normalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return perPage, (page - 1) * perPage
}
