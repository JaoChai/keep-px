package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

type NotificationHandler struct {
	notifService *service.NotificationService
}

func NewNotificationHandler(notifService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifService: notifService}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	result, err := h.notifService.List(r.Context(), customerID, limit)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: result})
}

func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	count, err := h.notifService.CountUnread(r.Context(), customerID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to count unread notifications")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: map[string]int{"unread_count": count}})
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.notifService.MarkRead(r.Context(), id, customerID); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to mark notification as read")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Message: "notification marked as read"})
}

func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	if err := h.notifService.MarkAllRead(r.Context(), customerID); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to mark all notifications as read")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Message: "all notifications marked as read"})
}
