package service

import (
	"context"
	"fmt"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

type NotificationService struct {
	repo repository.NotificationRepository
}

func NewNotificationService(repo repository.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

type NotificationListResult struct {
	Notifications []*domain.Notification `json:"notifications"`
	UnreadCount   int                    `json:"unread_count"`
}

func (s *NotificationService) CreateNotification(ctx context.Context, n *domain.Notification) error {
	if err := s.repo.Create(ctx, n); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

func (s *NotificationService) List(ctx context.Context, customerID string, limit int) (*NotificationListResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	notifications, err := s.repo.ListByCustomerID(ctx, customerID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	if notifications == nil {
		notifications = []*domain.Notification{}
	}

	unreadCount, err := s.repo.CountUnread(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("count unread notifications: %w", err)
	}

	return &NotificationListResult{
		Notifications: notifications,
		UnreadCount:   unreadCount,
	}, nil
}

func (s *NotificationService) CountUnread(ctx context.Context, customerID string) (int, error) {
	count, err := s.repo.CountUnread(ctx, customerID)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, id, customerID string) error {
	if err := s.repo.MarkRead(ctx, id, customerID); err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	return nil
}

func (s *NotificationService) MarkAllRead(ctx context.Context, customerID string) error {
	if err := s.repo.MarkAllRead(ctx, customerID); err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}
