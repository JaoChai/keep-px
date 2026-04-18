package handler

import (
	"net/http"
)

func (h *AdminHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	pixelID := r.URL.Query().Get("pixel_id")
	eventName := r.URL.Query().Get("event_name")
	page := queryInt(r, "page", 1)
	perPage := queryPerPage(r)

	events, total, err := h.adminService.ListAllEvents(r.Context(), customerID, pixelID, eventName, page, perPage)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to list events", err)
		return
	}

	h.paginatedResponse(w, events, total, page, perPage)
}

func (h *AdminHandler) GetEventStats(w http.ResponseWriter, r *http.Request) {
	hours := queryInt(r, "hours", 24)

	stats, err := h.adminService.GetEventStats(r.Context(), hours)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get event stats", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: stats})
}
