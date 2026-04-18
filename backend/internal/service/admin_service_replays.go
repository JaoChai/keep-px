package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

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
