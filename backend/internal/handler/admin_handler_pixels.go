package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

func (h *AdminHandler) ListPixels(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	customerID := r.URL.Query().Get("customer_id")
	active := queryBool(r, "active")
	page := queryInt(r, "page", 1)
	perPage := queryPerPage(r)

	pixels, total, err := h.adminService.ListAllPixels(r.Context(), search, customerID, active, page, perPage)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to list pixels", err)
		return
	}

	h.paginatedResponse(w, pixels, total, page, perPage)
}

func (h *AdminHandler) GetPixelDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	detail, err := h.adminService.GetPixelDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusNotFound, "pixel not found")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get pixel detail", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: detail})
}

func (h *AdminHandler) DisablePixel(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.adminService.DisablePixel(r.Context(), adminID, id); err != nil {
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusNotFound, "pixel not found")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to disable pixel", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "pixel disabled"})
}

func (h *AdminHandler) EnablePixel(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.adminService.EnablePixel(r.Context(), adminID, id); err != nil {
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusNotFound, "pixel not found")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to enable pixel", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "pixel enabled"})
}
