package handler

import (
	"net/http"
	"time"
)

func (h *AdminHandler) ListAuditLog(w http.ResponseWriter, r *http.Request) {
	adminID := r.URL.Query().Get("admin_id")
	action := r.URL.Query().Get("action")
	targetCustomerID := r.URL.Query().Get("target_customer_id")
	page := queryInt(r, "page", 1)
	perPage := queryPerPage(r)

	var from, to *time.Time
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = &t
		}
	}

	entries, total, err := h.adminService.ListAuditLogs(r.Context(), adminID, action, targetCustomerID, from, to, page, perPage)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to list audit log", err)
		return
	}

	h.paginatedResponse(w, entries, total, page, perPage)
}
