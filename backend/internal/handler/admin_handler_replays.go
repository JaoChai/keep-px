package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

func (h *AdminHandler) ListReplays(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	customerID := r.URL.Query().Get("customer_id")
	page := queryInt(r, "page", 1)
	perPage := queryPerPage(r)

	sessions, total, err := h.adminService.ListAllReplaySessions(r.Context(), status, customerID, page, perPage)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to list replays", err)
		return
	}

	h.paginatedResponse(w, sessions, total, page, perPage)
}

func (h *AdminHandler) GetReplayDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	detail, err := h.adminService.GetReplayDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrReplayNotFound) {
			ErrorJSON(w, http.StatusNotFound, "replay session not found")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get replay detail", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: detail})
}

func (h *AdminHandler) CancelReplay(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.adminService.CancelReplay(r.Context(), adminID, id); err != nil {
		if errors.Is(err, service.ErrReplayNotFound) {
			ErrorJSON(w, http.StatusNotFound, "replay session not found")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to cancel replay", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "replay cancelled"})
}
