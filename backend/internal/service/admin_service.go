package service

import (
	"context"
	"errors"
	"fmt"
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
	adminRepo    repository.AdminRepository
	customerRepo repository.CustomerRepository
	creditRepo   repository.ReplayCreditRepository
	pool         *pgxpool.Pool
}

func NewAdminService(
	adminRepo repository.AdminRepository,
	customerRepo repository.CustomerRepository,
	creditRepo repository.ReplayCreditRepository,
	pool *pgxpool.Pool,
) *AdminService {
	return &AdminService{
		adminRepo:    adminRepo,
		customerRepo: customerRepo,
		creditRepo:   creditRepo,
		pool:         pool,
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

func (s *AdminService) ChangePlan(ctx context.Context, customerID, newPlan string) error {
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

	return s.customerRepo.UpdatePlan(ctx, customerID, newPlan)
}

func (s *AdminService) SuspendCustomer(ctx context.Context, adminID, customerID string) error {
	if adminID == customerID {
		return ErrAdminSelfSuspend
	}
	err := s.adminRepo.SuspendCustomer(ctx, customerID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrCustomerNotFound
	}
	return err
}

func (s *AdminService) ActivateCustomer(ctx context.Context, customerID string) error {
	err := s.adminRepo.ActivateCustomer(ctx, customerID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrCustomerNotFound
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
	defer tx.Rollback(ctx)

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
