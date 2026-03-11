package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

type AnalyticsHandler struct {
	analyticsService *service.AnalyticsService
	logger           *slog.Logger
}

func NewAnalyticsHandler(analyticsService *service.AnalyticsService, logger *slog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService, logger: logger}
}

func (h *AnalyticsHandler) Overview(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	stats, err := h.analyticsService.GetOverview(r.Context(), customerID)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get overview", err)
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: stats})
}

func (h *AnalyticsHandler) EventChart(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days < 1 {
		days = 30
	}

	data, err := h.analyticsService.GetEventChart(r.Context(), customerID, days)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get event chart", err)
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: data})
}
