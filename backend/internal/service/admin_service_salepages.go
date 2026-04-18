package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

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
