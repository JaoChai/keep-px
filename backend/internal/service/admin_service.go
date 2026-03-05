package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

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

func (s *AdminService) ListCustomers(ctx context.Context, search, plan, status string, page, perPage int) ([]*domain.Customer, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	// Validate plan against allow-list
	if plan != "" {
		if _, ok := domain.PlanLimitsMap[plan]; !ok {
			return nil, 0, ErrInvalidPlan
		}
	}

	// Validate status against allow-list
	if status != "" && status != "active" && status != "suspended" {
		status = ""
	}

	return s.adminRepo.ListCustomers(ctx, search, plan, status, perPage, offset)
}

func (s *AdminService) GetCustomerDetail(ctx context.Context, customerID string) (*domain.AdminCustomerDetail, error) {
	detail, err := s.adminRepo.GetCustomerDetail(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer detail: %w", err)
	}
	if detail == nil {
		return nil, ErrCustomerNotFound
	}
	return detail, nil
}

func (s *AdminService) ChangePlan(ctx context.Context, adminID, customerID, newPlan string) error {
	if _, ok := domain.PlanLimitsMap[newPlan]; !ok {
		return ErrInvalidPlan
	}

	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return ErrCustomerNotFound
	}

	if err := s.customerRepo.UpdatePlan(ctx, customerID, newPlan); err != nil {
		return err
	}
	s.logAudit(ctx, adminID, "change_plan", "customer", customerID, &customerID, map[string]string{"new_plan": newPlan})
	return nil
}

func (s *AdminService) SuspendCustomer(ctx context.Context, adminID, customerID string) error {
	if adminID == customerID {
		return ErrAdminSelfSuspend
	}
	err := s.adminRepo.SuspendCustomer(ctx, customerID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrCustomerNotFound
	}
	if err == nil {
		s.logAudit(ctx, adminID, "suspend_customer", "customer", customerID, &customerID, nil)
	}
	return err
}

func (s *AdminService) ActivateCustomer(ctx context.Context, adminID, customerID string) error {
	err := s.adminRepo.ActivateCustomer(ctx, customerID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrCustomerNotFound
	}
	if err == nil {
		s.logAudit(ctx, adminID, "activate_customer", "customer", customerID, &customerID, nil)
	}
	return err
}

type GrantCreditsInput struct {
	PackType           string `json:"pack_type" validate:"required"`
	TotalReplays       int    `json:"total_replays" validate:"required,min=1"`
	MaxEventsPerReplay int    `json:"max_events_per_replay" validate:"required,min=1"`
	ExpiryDays         int    `json:"expiry_days" validate:"required,min=1"`
	Reason             string `json:"reason"`
}

func (s *AdminService) GrantCredits(ctx context.Context, adminID, customerID string, input GrantCreditsInput) (*domain.AdminCreditGrant, error) {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil, ErrCustomerNotFound
	}

	expiresAt := time.Now().AddDate(0, 0, input.ExpiryDays)

	// Use transaction to ensure credit + audit record are atomic
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	// Create the replay credit
	credit := &domain.ReplayCredit{
		CustomerID:         customerID,
		PackType:           input.PackType,
		TotalReplays:       input.TotalReplays,
		UsedReplays:        0,
		MaxEventsPerReplay: input.MaxEventsPerReplay,
		ExpiresAt:          expiresAt,
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO replay_credits (customer_id, pack_type, total_replays, used_replays, max_events_per_replay, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		credit.CustomerID, credit.PackType, credit.TotalReplays, credit.UsedReplays, credit.MaxEventsPerReplay, credit.ExpiresAt,
	).Scan(&credit.ID, &credit.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create replay credit: %w", err)
	}

	// Create the audit record
	grant := &domain.AdminCreditGrant{
		AdminID:            adminID,
		CustomerID:         customerID,
		PackType:           input.PackType,
		TotalReplays:       input.TotalReplays,
		MaxEventsPerReplay: input.MaxEventsPerReplay,
		ExpiresAt:          expiresAt,
		Reason:             input.Reason,
		CreditID:           &credit.ID,
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO admin_credit_grants (admin_id, customer_id, pack_type, total_replays, max_events_per_replay, expires_at, reason, credit_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at`,
		grant.AdminID, grant.CustomerID, grant.PackType, grant.TotalReplays, grant.MaxEventsPerReplay, grant.ExpiresAt, grant.Reason, grant.CreditID,
	).Scan(&grant.ID, &grant.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create credit grant audit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	s.logAudit(ctx, adminID, "grant_credits", "customer", customerID, &customerID, map[string]interface{}{
		"pack_type":      input.PackType,
		"total_replays":  input.TotalReplays,
		"credit_id":      credit.ID,
	})

	return grant, nil
}

func (s *AdminService) GetPlatformOverview(ctx context.Context) (*domain.PlatformStats, error) {
	return s.adminRepo.GetPlatformStats(ctx)
}

func (s *AdminService) GetRevenueChart(ctx context.Context, days int) ([]*domain.RevenueChartPoint, error) {
	if days < 1 || days > 365 {
		days = 30
	}
	return s.adminRepo.GetRevenueChart(ctx, days)
}

func (s *AdminService) GetGrowthChart(ctx context.Context, days int) ([]*domain.GrowthChartPoint, error) {
	if days < 1 || days > 365 {
		days = 30
	}
	return s.adminRepo.GetGrowthChart(ctx, days)
}

func (s *AdminService) ListAllPurchases(ctx context.Context, status string, page, perPage int) ([]*domain.AdminPurchase, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	return s.adminRepo.ListAllPurchases(ctx, status, perPage, offset)
}

func (s *AdminService) ListAllSubscriptions(ctx context.Context, status string, page, perPage int) ([]*domain.AdminSubscription, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	return s.adminRepo.ListAllSubscriptions(ctx, status, perPage, offset)
}

func (s *AdminService) ListCreditGrants(ctx context.Context, page, perPage int) ([]*domain.AdminCreditGrantWithCustomer, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	return s.adminRepo.ListCreditGrants(ctx, perPage, offset)
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

// F1: Sale Pages

func (s *AdminService) ListAllSalePages(ctx context.Context, search, customerID string, published *bool, page, perPage int) ([]*domain.AdminSalePage, int, error) {
	perPage, offset := s.normalizePagination(page, perPage)
	return s.adminRepo.ListAllSalePages(ctx, search, customerID, published, perPage, offset)
}

func (s *AdminService) GetSalePageDetail(ctx context.Context, id string) (*domain.AdminSalePageDetail, error) {
	detail, err := s.adminRepo.GetSalePageAdminDetail(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get sale page detail: %w", err)
	}
	if detail == nil {
		return nil, ErrSalePageNotFound
	}
	return detail, nil
}

func (s *AdminService) DisableSalePage(ctx context.Context, adminID, id string) error {
	err := s.adminRepo.SetSalePagePublished(ctx, id, false)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrSalePageNotFound
	}
	if err != nil {
		return err
	}
	s.logAudit(ctx, adminID, "disable_sale_page", "sale_page", id, nil, nil)
	return nil
}

func (s *AdminService) EnableSalePage(ctx context.Context, adminID, id string) error {
	err := s.adminRepo.SetSalePagePublished(ctx, id, true)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrSalePageNotFound
	}
	if err != nil {
		return err
	}
	s.logAudit(ctx, adminID, "enable_sale_page", "sale_page", id, nil, nil)
	return nil
}

func (s *AdminService) DeleteSalePage(ctx context.Context, adminID, id string) error {
	detail, err := s.adminRepo.GetSalePageAdminDetail(ctx, id)
	if err != nil {
		return fmt.Errorf("get sale page for delete: %w", err)
	}
	if detail == nil {
		return ErrSalePageNotFound
	}

	if err := s.adminRepo.DeleteSalePageByAdmin(ctx, id); err != nil {
		return err
	}

	custID := detail.SalePage.CustomerID
	s.logAudit(ctx, adminID, "delete_sale_page", "sale_page", id, &custID, nil)
	return nil
}

// F2: Pixels

func (s *AdminService) ListAllPixels(ctx context.Context, search, customerID string, active *bool, page, perPage int) ([]*domain.AdminPixel, int, error) {
	perPage, offset := s.normalizePagination(page, perPage)
	return s.adminRepo.ListAllPixels(ctx, search, customerID, active, perPage, offset)
}

func (s *AdminService) GetPixelDetail(ctx context.Context, id string) (*domain.AdminPixelDetail, error) {
	detail, err := s.adminRepo.GetPixelAdminDetail(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get pixel detail: %w", err)
	}
	if detail == nil {
		return nil, ErrPixelNotFound
	}
	return detail, nil
}

func (s *AdminService) DisablePixel(ctx context.Context, adminID, id string) error {
	err := s.adminRepo.SetPixelActive(ctx, id, false)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrPixelNotFound
	}
	if err != nil {
		return err
	}
	s.logAudit(ctx, adminID, "disable_pixel", "pixel", id, nil, nil)
	return nil
}

func (s *AdminService) EnablePixel(ctx context.Context, adminID, id string) error {
	err := s.adminRepo.SetPixelActive(ctx, id, true)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrPixelNotFound
	}
	if err != nil {
		return err
	}
	s.logAudit(ctx, adminID, "enable_pixel", "pixel", id, nil, nil)
	return nil
}

// F3: Replays

func (s *AdminService) ListAllReplaySessions(ctx context.Context, status, customerID string, page, perPage int) ([]*domain.AdminReplaySession, int, error) {
	perPage, offset := s.normalizePagination(page, perPage)
	return s.adminRepo.ListAllReplaySessions(ctx, status, customerID, perPage, offset)
}

func (s *AdminService) GetReplayDetail(ctx context.Context, id string) (*domain.AdminReplaySessionDetail, error) {
	detail, err := s.adminRepo.GetReplaySessionAdminDetail(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get replay detail: %w", err)
	}
	if detail == nil {
		return nil, ErrReplayNotFound
	}
	return detail, nil
}

func (s *AdminService) CancelReplay(ctx context.Context, adminID, id string) error {
	session, err := s.replaySessionRepo.CancelSession(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrReplayNotFound
	}
	if err != nil {
		return err
	}
	custID := session.CustomerID
	s.logAudit(ctx, adminID, "cancel_replay", "replay_session", id, &custID, nil)
	return nil
}

// F4: Events

func (s *AdminService) ListAllEvents(ctx context.Context, customerID, pixelID, eventName string, page, perPage int) ([]*domain.AdminEvent, int, error) {
	perPage, offset := s.normalizePagination(page, perPage)
	return s.adminRepo.ListAllEvents(ctx, customerID, pixelID, eventName, perPage, offset)
}

func (s *AdminService) GetEventStats(ctx context.Context, hours int) (*domain.AdminEventStats, error) {
	if hours < 1 || hours > 72 {
		hours = 24
	}
	return s.adminRepo.GetEventStats(ctx, hours)
}

// F5: Audit Log

var validAuditActions = map[string]bool{
	"suspend_customer":  true,
	"activate_customer": true,
	"change_plan":       true,
	"grant_credits":     true,
	"disable_sale_page": true,
	"enable_sale_page":  true,
	"delete_sale_page":  true,
	"disable_pixel":     true,
	"enable_pixel":      true,
	"cancel_replay":     true,
}

func (s *AdminService) ListAuditLogs(ctx context.Context, adminID, action, targetCustomerID string, from, to *time.Time, page, perPage int) ([]*domain.AuditLogEntry, int, error) {
	if action != "" && !validAuditActions[action] {
		action = ""
	}
	perPage, offset := s.normalizePagination(page, perPage)
	return s.adminRepo.ListAuditLogs(ctx, adminID, action, targetCustomerID, from, to, perPage, offset)
}
