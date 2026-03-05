package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

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
	adminID := middleware.GetCustomerID(r.Context())
	customerID := chi.URLParam(r, "id")

	if err := h.adminService.ActivateCustomer(r.Context(), adminID, customerID); err != nil {
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

func queryBool(r *http.Request, key string) *bool {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil
	}
	b := v == "true" || v == "1"
	return &b
}

func (h *AdminHandler) paginatedResponse(w http.ResponseWriter, data interface{}, total, page, perPage int) {
	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}
	JSON(w, http.StatusOK, PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

// F1: Sale Pages

func (h *AdminHandler) ListSalePages(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	customerID := r.URL.Query().Get("customer_id")
	published := queryBool(r, "published")
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "per_page", 20)

	pages, total, err := h.adminService.ListAllSalePages(r.Context(), search, customerID, published, page, perPage)
	if err != nil {
		h.logger.Error("list sale pages failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to list sale pages")
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
		h.logger.Error("get sale page detail failed", "error", err, "id", id)
		ErrorJSON(w, http.StatusInternalServerError, "failed to get sale page detail")
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
		h.logger.Error("disable sale page failed", "error", err, "id", id)
		ErrorJSON(w, http.StatusInternalServerError, "failed to disable sale page")
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
		h.logger.Error("enable sale page failed", "error", err, "id", id)
		ErrorJSON(w, http.StatusInternalServerError, "failed to enable sale page")
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
		h.logger.Error("delete sale page failed", "error", err, "id", id)
		ErrorJSON(w, http.StatusInternalServerError, "failed to delete sale page")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "sale page deleted"})
}

// F2: Pixels

func (h *AdminHandler) ListPixels(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	customerID := r.URL.Query().Get("customer_id")
	active := queryBool(r, "active")
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "per_page", 20)

	pixels, total, err := h.adminService.ListAllPixels(r.Context(), search, customerID, active, page, perPage)
	if err != nil {
		h.logger.Error("list pixels failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to list pixels")
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
		h.logger.Error("get pixel detail failed", "error", err, "id", id)
		ErrorJSON(w, http.StatusInternalServerError, "failed to get pixel detail")
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
		h.logger.Error("disable pixel failed", "error", err, "id", id)
		ErrorJSON(w, http.StatusInternalServerError, "failed to disable pixel")
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
		h.logger.Error("enable pixel failed", "error", err, "id", id)
		ErrorJSON(w, http.StatusInternalServerError, "failed to enable pixel")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "pixel enabled"})
}

// F3: Replays

func (h *AdminHandler) ListReplays(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	customerID := r.URL.Query().Get("customer_id")
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "per_page", 20)

	sessions, total, err := h.adminService.ListAllReplaySessions(r.Context(), status, customerID, page, perPage)
	if err != nil {
		h.logger.Error("list replays failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to list replays")
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
		h.logger.Error("get replay detail failed", "error", err, "id", id)
		ErrorJSON(w, http.StatusInternalServerError, "failed to get replay detail")
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
		h.logger.Error("cancel replay failed", "error", err, "id", id)
		ErrorJSON(w, http.StatusInternalServerError, "failed to cancel replay")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "replay cancelled"})
}

// F4: Events

func (h *AdminHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	pixelID := r.URL.Query().Get("pixel_id")
	eventName := r.URL.Query().Get("event_name")
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "per_page", 20)

	events, total, err := h.adminService.ListAllEvents(r.Context(), customerID, pixelID, eventName, page, perPage)
	if err != nil {
		h.logger.Error("list events failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to list events")
		return
	}

	h.paginatedResponse(w, events, total, page, perPage)
}

func (h *AdminHandler) GetEventStats(w http.ResponseWriter, r *http.Request) {
	hours := queryInt(r, "hours", 24)

	stats, err := h.adminService.GetEventStats(r.Context(), hours)
	if err != nil {
		h.logger.Error("get event stats failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to get event stats")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: stats})
}

// F5: Audit Log

func (h *AdminHandler) ListAuditLog(w http.ResponseWriter, r *http.Request) {
	adminID := r.URL.Query().Get("admin_id")
	action := r.URL.Query().Get("action")
	targetCustomerID := r.URL.Query().Get("target_customer_id")
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "per_page", 20)

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
		h.logger.Error("list audit log failed", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to list audit log")
		return
	}

	h.paginatedResponse(w, entries, total, page, perPage)
}
