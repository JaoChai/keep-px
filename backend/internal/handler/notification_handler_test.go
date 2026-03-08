package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

// ---------------------------------------------------------------------------
// Notification handler integration tests
// ---------------------------------------------------------------------------

func TestNotificationHandler_List(t *testing.T) {
	t.Run("success returns 200 with notifications", func(t *testing.T) {
		notifRepo := &MockNotificationRepo{}
		svc := newTestNotificationService(notifRepo)
		h := NewNotificationHandler(svc)

		now := time.Now()
		notifs := []*domain.Notification{
			{ID: "n1", CustomerID: "cust-1", Type: domain.NotificationTypeSystem, Title: "Welcome", Body: "Hello", CreatedAt: now},
			{ID: "n2", CustomerID: "cust-1", Type: domain.NotificationTypeReplayCompleted, Title: "Replay done", Body: "Your replay completed", CreatedAt: now},
		}

		notifRepo.On("ListByCustomerID", mock.Anything, "cust-1", 20).Return(notifs, nil)
		notifRepo.On("CountUnread", mock.Anything, "cust-1").Return(1, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Get("/notifications", h.List)

		rec := doRequest(r, "GET", "/notifications", nil, testJWT("cust-1", false))

		assert.Equal(t, 200, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Data)

		notifRepo.AssertExpectations(t)
	})

	t.Run("with limit query param", func(t *testing.T) {
		notifRepo := &MockNotificationRepo{}
		svc := newTestNotificationService(notifRepo)
		h := NewNotificationHandler(svc)

		notifRepo.On("ListByCustomerID", mock.Anything, "cust-1", 5).Return([]*domain.Notification{}, nil)
		notifRepo.On("CountUnread", mock.Anything, "cust-1").Return(0, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Get("/notifications", h.List)

		rec := doRequest(r, "GET", "/notifications?limit=5", nil, testJWT("cust-1", false))

		assert.Equal(t, 200, rec.Code)
		notifRepo.AssertExpectations(t)
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		notifRepo := &MockNotificationRepo{}
		svc := newTestNotificationService(notifRepo)
		h := NewNotificationHandler(svc)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Get("/notifications", h.List)

		rec := doRequest(r, "GET", "/notifications", nil, "")

		assert.Equal(t, 401, rec.Code)
	})
}

func TestNotificationHandler_UnreadCount(t *testing.T) {
	t.Run("success returns 200 with unread count", func(t *testing.T) {
		notifRepo := &MockNotificationRepo{}
		svc := newTestNotificationService(notifRepo)
		h := NewNotificationHandler(svc)

		notifRepo.On("CountUnread", mock.Anything, "cust-1").Return(7, nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Get("/notifications/unread-count", h.UnreadCount)

		rec := doRequest(r, "GET", "/notifications/unread-count", nil, testJWT("cust-1", false))

		assert.Equal(t, 200, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		dataMap, ok := resp.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(7), dataMap["unread_count"])

		notifRepo.AssertExpectations(t)
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		notifRepo := &MockNotificationRepo{}
		svc := newTestNotificationService(notifRepo)
		h := NewNotificationHandler(svc)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Get("/notifications/unread-count", h.UnreadCount)

		rec := doRequest(r, "GET", "/notifications/unread-count", nil, "")

		assert.Equal(t, 401, rec.Code)
	})
}

func TestNotificationHandler_MarkRead(t *testing.T) {
	t.Run("success returns 200", func(t *testing.T) {
		notifRepo := &MockNotificationRepo{}
		svc := newTestNotificationService(notifRepo)
		h := NewNotificationHandler(svc)

		notifRepo.On("MarkRead", mock.Anything, "notif-1", "cust-1").Return(nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Put("/notifications/{id}/read", h.MarkRead)

		rec := doRequest(r, "PUT", "/notifications/notif-1/read", nil, testJWT("cust-1", false))

		assert.Equal(t, 200, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "notification marked as read", resp.Message)

		notifRepo.AssertExpectations(t)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		notifRepo := &MockNotificationRepo{}
		svc := newTestNotificationService(notifRepo)
		h := NewNotificationHandler(svc)

		notifRepo.On("MarkRead", mock.Anything, "nonexistent", "cust-1").Return(repository.ErrNotFound)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Put("/notifications/{id}/read", h.MarkRead)

		rec := doRequest(r, "PUT", "/notifications/nonexistent/read", nil, testJWT("cust-1", false))

		assert.Equal(t, 404, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "notification not found", resp.Error)

		notifRepo.AssertExpectations(t)
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		notifRepo := &MockNotificationRepo{}
		svc := newTestNotificationService(notifRepo)
		h := NewNotificationHandler(svc)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Put("/notifications/{id}/read", h.MarkRead)

		rec := doRequest(r, "PUT", "/notifications/notif-1/read", nil, "")

		assert.Equal(t, 401, rec.Code)
	})
}

func TestNotificationHandler_MarkAllRead(t *testing.T) {
	t.Run("success returns 200", func(t *testing.T) {
		notifRepo := &MockNotificationRepo{}
		svc := newTestNotificationService(notifRepo)
		h := NewNotificationHandler(svc)

		notifRepo.On("MarkAllRead", mock.Anything, "cust-1").Return(nil)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Put("/notifications/read-all", h.MarkAllRead)

		rec := doRequest(r, "PUT", "/notifications/read-all", nil, testJWT("cust-1", false))

		assert.Equal(t, 200, rec.Code)

		var resp APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "all notifications marked as read", resp.Message)

		notifRepo.AssertExpectations(t)
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		notifRepo := &MockNotificationRepo{}
		svc := newTestNotificationService(notifRepo)
		h := NewNotificationHandler(svc)

		r := chi.NewRouter()
		r.Use(middleware.JWTAuth(testJWTSecret))
		r.Put("/notifications/read-all", h.MarkAllRead)

		rec := doRequest(r, "PUT", "/notifications/read-all", nil, "")

		assert.Equal(t, 401, rec.Code)
	})
}
