package service

import (
	"context"
	"time"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

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
