package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

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
