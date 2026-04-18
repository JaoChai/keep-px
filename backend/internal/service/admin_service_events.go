package service

import (
	"context"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

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
