package handler

import (
	"net/http"
)

func (h *AdminHandler) ListPurchases(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	page := queryInt(r, "page", 1)
	perPage := queryPerPage(r)

	purchases, total, err := h.adminService.ListAllPurchases(r.Context(), status, page, perPage)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to list purchases", err)
		return
	}

	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}

	JSON(w, http.StatusOK, PaginatedResponse{
		Data:       purchases,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

func (h *AdminHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	page := queryInt(r, "page", 1)
	perPage := queryPerPage(r)

	subs, total, err := h.adminService.ListAllSubscriptions(r.Context(), status, page, perPage)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to list subscriptions", err)
		return
	}

	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}

	JSON(w, http.StatusOK, PaginatedResponse{
		Data:       subs,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

func (h *AdminHandler) ListCreditGrants(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	perPage := queryPerPage(r)

	grants, total, err := h.adminService.ListCreditGrants(r.Context(), page, perPage)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to list credit grants", err)
		return
	}

	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}

	JSON(w, http.StatusOK, PaginatedResponse{
		Data:       grants,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}
