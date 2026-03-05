package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

type AdminHandler struct {
	adminService *service.AdminService
	validate     *validator.Validate
	logger       *slog.Logger
}

func NewAdminHandler(adminService *service.AdminService, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
		validate:     validator.New(),
		logger:       logger,
	}
}

func (h *AdminHandler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	plan := r.URL.Query().Get("plan")
	status := r.URL.Query().Get("status")
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "per_page", 20)

	customers, total, err := h.adminService.ListCustomers(r.Context(), search, plan, status, page, perPage)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPlan) {
			ErrorJSON(w, http.StatusBadRequest, "invalid plan filter")
			return
		}
		h.logger.Error("list customers failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to list customers")
		return
	}

	// Strip sensitive data — admin list should not expose API keys
	for _, c := range customers {
		c.APIKey = ""
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
		h.logger.Error("get customer detail failed", "error", err, "customer_id", customerID)
		ErrorJSON(w, http.StatusInternalServerError, "failed to get customer detail")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: detail})
}

func (h *AdminHandler) ChangePlan(w http.ResponseWriter, r *http.Request) {
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

	if err := h.adminService.ChangePlan(r.Context(), customerID, input.Plan); err != nil {
		switch {
		case errors.Is(err, service.ErrCustomerNotFound):
			ErrorJSON(w, http.StatusNotFound, "customer not found")
		case errors.Is(err, service.ErrInvalidPlan):
			ErrorJSON(w, http.StatusBadRequest, "invalid plan")
		default:
			h.logger.Error("change plan failed", "error", err, "customer_id", customerID)
			ErrorJSON(w, http.StatusInternalServerError, "failed to change plan")
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
			h.logger.Error("suspend customer failed", "error", err, "customer_id", customerID)
			ErrorJSON(w, http.StatusInternalServerError, "failed to suspend customer")
		}
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "customer suspended"})
}

func (h *AdminHandler) ActivateCustomer(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")

	if err := h.adminService.ActivateCustomer(r.Context(), customerID); err != nil {
		if errors.Is(err, service.ErrCustomerNotFound) {
			ErrorJSON(w, http.StatusNotFound, "customer not found")
			return
		}
		h.logger.Error("activate customer failed", "error", err, "customer_id", customerID)
		ErrorJSON(w, http.StatusInternalServerError, "failed to activate customer")
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
		h.logger.Error("grant credits failed", "error", err, "customer_id", customerID)
		ErrorJSON(w, http.StatusInternalServerError, "failed to grant credits")
		return
	}

	JSON(w, http.StatusCreated, APIResponse{Data: grant})
}

func (h *AdminHandler) GetPlatformOverview(w http.ResponseWriter, r *http.Request) {
	stats, err := h.adminService.GetPlatformOverview(r.Context())
	if err != nil {
		h.logger.Error("get platform overview failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to get platform overview")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: stats})
}

func (h *AdminHandler) GetRevenueChart(w http.ResponseWriter, r *http.Request) {
	days := queryInt(r, "days", 30)

	chart, err := h.adminService.GetRevenueChart(r.Context(), days)
	if err != nil {
		h.logger.Error("get revenue chart failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to get revenue chart")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: chart})
}

func (h *AdminHandler) GetGrowthChart(w http.ResponseWriter, r *http.Request) {
	days := queryInt(r, "days", 30)

	chart, err := h.adminService.GetGrowthChart(r.Context(), days)
	if err != nil {
		h.logger.Error("get growth chart failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to get growth chart")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: chart})
}

func (h *AdminHandler) ListPurchases(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "per_page", 20)

	purchases, total, err := h.adminService.ListAllPurchases(r.Context(), status, page, perPage)
	if err != nil {
		h.logger.Error("list purchases failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to list purchases")
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
	perPage := queryInt(r, "per_page", 20)

	subs, total, err := h.adminService.ListAllSubscriptions(r.Context(), status, page, perPage)
	if err != nil {
		h.logger.Error("list subscriptions failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to list subscriptions")
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
	perPage := queryInt(r, "per_page", 20)

	grants, total, err := h.adminService.ListCreditGrants(r.Context(), page, perPage)
	if err != nil {
		h.logger.Error("list credit grants failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to list credit grants")
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

func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(v)
	if err != nil || i < 1 {
		return defaultVal
	}
	return i
}
