package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

type GrantCreditsInput struct {
	PackType           string `json:"pack_type" validate:"required"`
	TotalReplays       int    `json:"total_replays" validate:"required,min=1"`
	MaxEventsPerReplay int    `json:"max_events_per_replay" validate:"required,min=1"`
	ExpiryDays         int    `json:"expiry_days" validate:"required,min=1"`
	Reason             string `json:"reason"`
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
	if plan != "" && plan != domain.PlanSandbox && plan != domain.PlanPaid {
		return nil, 0, ErrInvalidPlan
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
	if newPlan != domain.PlanSandbox && newPlan != domain.PlanPaid {
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
		"pack_type":     input.PackType,
		"total_replays": input.TotalReplays,
		"credit_id":     credit.ID,
	})

	return grant, nil
}
