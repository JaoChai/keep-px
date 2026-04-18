package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

func (h *AdminHandler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	plan := r.URL.Query().Get("plan")
	status := r.URL.Query().Get("status")
	page := queryInt(r, "page", 1)
	perPage := queryPerPage(r)

	customers, total, err := h.adminService.ListCustomers(r.Context(), search, plan, status, page, perPage)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPlan) {
			ErrorJSON(w, http.StatusBadRequest, "invalid plan filter")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to list customers", err)
		return
	}

	// Strip sensitive data — admin list should not expose API keys or password hashes
	for _, c := range customers {
		c.APIKey = ""
		c.PasswordHash = ""
	}

	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}

	JSON(w, http.StatusOK, PaginatedResponse{
		Data:       customers,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

func (h *AdminHandler) GetCustomerDetail(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")

	detail, err := h.adminService.GetCustomerDetail(r.Context(), customerID)
	if err != nil {
		if errors.Is(err, service.ErrCustomerNotFound) {
			ErrorJSON(w, http.StatusNotFound, "customer not found")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get customer detail", err)
		return
	}

	// Strip sensitive data
	detail.Customer.PasswordHash = ""

	JSON(w, http.StatusOK, APIResponse{Data: detail})
}

func (h *AdminHandler) ChangePlan(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetCustomerID(r.Context())
	customerID := chi.URLParam(r, "id")

	var input struct {
		Plan string `json:"plan" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, FormatValidationErrors(err))
		return
	}

	if err := h.adminService.ChangePlan(r.Context(), adminID, customerID, input.Plan); err != nil {
		switch {
		case errors.Is(err, service.ErrCustomerNotFound):
			ErrorJSON(w, http.StatusNotFound, "customer not found")
		case errors.Is(err, service.ErrInvalidPlan):
			ErrorJSON(w, http.StatusBadRequest, "invalid plan")
		default:
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to change plan", err)
		}
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "plan updated"})
}

func (h *AdminHandler) SuspendCustomer(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetCustomerID(r.Context())
	customerID := chi.URLParam(r, "id")

	if err := h.adminService.SuspendCustomer(r.Context(), adminID, customerID); err != nil {
		switch {
		case errors.Is(err, service.ErrAdminSelfSuspend):
			ErrorJSON(w, http.StatusBadRequest, "cannot suspend your own account")
		case errors.Is(err, service.ErrCustomerNotFound):
			ErrorJSON(w, http.StatusNotFound, "customer not found")
		default:
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to suspend customer", err)
		}
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "customer suspended"})
}

func (h *AdminHandler) ActivateCustomer(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetCustomerID(r.Context())
	customerID := chi.URLParam(r, "id")

	if err := h.adminService.ActivateCustomer(r.Context(), adminID, customerID); err != nil {
		if errors.Is(err, service.ErrCustomerNotFound) {
			ErrorJSON(w, http.StatusNotFound, "customer not found")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to activate customer", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "customer activated"})
}

func (h *AdminHandler) GrantCredits(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetCustomerID(r.Context())
	customerID := chi.URLParam(r, "id")

	var input service.GrantCreditsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, FormatValidationErrors(err))
		return
	}

	grant, err := h.adminService.GrantCredits(r.Context(), adminID, customerID, input)
	if err != nil {
		if errors.Is(err, service.ErrCustomerNotFound) {
			ErrorJSON(w, http.StatusNotFound, "customer not found")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to grant credits", err)
		return
	}

	JSON(w, http.StatusCreated, APIResponse{Data: grant})
}
