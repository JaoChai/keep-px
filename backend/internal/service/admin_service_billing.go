package service

import (
	"context"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

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
