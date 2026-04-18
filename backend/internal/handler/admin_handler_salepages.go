package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

func (h *AdminHandler) ListSalePages(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	customerID := r.URL.Query().Get("customer_id")
	published := queryBool(r, "published")
	page := queryInt(r, "page", 1)
	perPage := queryPerPage(r)

	pages, total, err := h.adminService.ListAllSalePages(r.Context(), search, customerID, published, page, perPage)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to list sale pages", err)
		return
	}

	h.paginatedResponse(w, pages, total, page, perPage)
}

func (h *AdminHandler) GetSalePageDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	detail, err := h.adminService.GetSalePageDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrSalePageNotFound) {
			ErrorJSON(w, http.StatusNotFound, "sale page not found")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get sale page detail", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: detail})
}

func (h *AdminHandler) DisableSalePage(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.adminService.DisableSalePage(r.Context(), adminID, id); err != nil {
		if errors.Is(err, service.ErrSalePageNotFound) {
			ErrorJSON(w, http.StatusNotFound, "sale page not found")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to disable sale page", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "sale page disabled"})
}

func (h *AdminHandler) EnableSalePage(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.adminService.EnableSalePage(r.Context(), adminID, id); err != nil {
		if errors.Is(err, service.ErrSalePageNotFound) {
			ErrorJSON(w, http.StatusNotFound, "sale page not found")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to enable sale page", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "sale page enabled"})
}

func (h *AdminHandler) DeleteSalePage(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.adminService.DeleteSalePage(r.Context(), adminID, id); err != nil {
		if errors.Is(err, service.ErrSalePageNotFound) {
			ErrorJSON(w, http.StatusNotFound, "sale page not found")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to delete sale page", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "sale page deleted"})
}
