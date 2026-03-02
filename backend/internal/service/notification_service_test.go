package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

func newTestNotificationService() (*NotificationService, *MockNotificationRepo) {
	repo := new(MockNotificationRepo)
	svc := NewNotificationService(repo)
	return svc, repo
}

func TestCreateNotification_Success(t *testing.T) {
	svc, repo := newTestNotificationService()

	n := &domain.Notification{
		CustomerID: "cust-1",
		Type:       domain.NotificationTypeReplayCompleted,
		Title:      "Replay completed",
		Body:       "Replayed 500 events",
		Metadata:   json.RawMessage(`{"session_id":"sess-1"}`),
	}

	repo.On("Create", mock.Anything, n).Run(func(args mock.Arguments) {
		notif := args.Get(1).(*domain.Notification)
		notif.ID = "notif-1"
		notif.CreatedAt = time.Now()
	}).Return(nil)

	err := svc.CreateNotification(context.Background(), n)
	assert.NoError(t, err)
	assert.Equal(t, "notif-1", n.ID)
	repo.AssertExpectations(t)
}

func TestList_ReturnsNotificationsWithUnreadCount(t *testing.T) {
	svc, repo := newTestNotificationService()

	notifications := []*domain.Notification{
		{ID: "n-1", CustomerID: "cust-1", Type: "system", Title: "Test", IsRead: false},
		{ID: "n-2", CustomerID: "cust-1", Type: "system", Title: "Test 2", IsRead: true},
	}

	repo.On("ListByCustomerID", mock.Anything, "cust-1", 20).Return(notifications, nil)
	repo.On("CountUnread", mock.Anything, "cust-1").Return(1, nil)

	result, err := svc.List(context.Background(), "cust-1", 20)
	assert.NoError(t, err)
	assert.Len(t, result.Notifications, 2)
	assert.Equal(t, 1, result.UnreadCount)
	repo.AssertExpectations(t)
}

func TestList_DefaultsLimitTo20(t *testing.T) {
	svc, repo := newTestNotificationService()

	repo.On("ListByCustomerID", mock.Anything, "cust-1", 20).Return([]*domain.Notification{}, nil)
	repo.On("CountUnread", mock.Anything, "cust-1").Return(0, nil)

	result, err := svc.List(context.Background(), "cust-1", 0)
	assert.NoError(t, err)
	assert.Empty(t, result.Notifications)
	assert.Equal(t, 0, result.UnreadCount)
	repo.AssertExpectations(t)
}

func TestCountUnread_ReturnsCount(t *testing.T) {
	svc, repo := newTestNotificationService()

	repo.On("CountUnread", mock.Anything, "cust-1").Return(5, nil)

	count, err := svc.CountUnread(context.Background(), "cust-1")
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
	repo.AssertExpectations(t)
}

func TestMarkRead_CallsRepo(t *testing.T) {
	svc, repo := newTestNotificationService()

	repo.On("MarkRead", mock.Anything, "notif-1", "cust-1").Return(nil)

	err := svc.MarkRead(context.Background(), "notif-1", "cust-1")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestMarkAllRead_CallsRepo(t *testing.T) {
	svc, repo := newTestNotificationService()

	repo.On("MarkAllRead", mock.Anything, "cust-1").Return(nil)

	err := svc.MarkAllRead(context.Background(), "cust-1")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
