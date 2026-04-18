package handler

import (
	"net/http"
)

func (h *AdminHandler) GetPlatformOverview(w http.ResponseWriter, r *http.Request) {
	stats, err := h.adminService.GetPlatformOverview(r.Context())
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get platform overview", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: stats})
}

func (h *AdminHandler) GetRevenueChart(w http.ResponseWriter, r *http.Request) {
	days := queryInt(r, "days", 30)

	chart, err := h.adminService.GetRevenueChart(r.Context(), days)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get revenue chart", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: chart})
}

func (h *AdminHandler) GetGrowthChart(w http.ResponseWriter, r *http.Request) {
	days := queryInt(r, "days", 30)

	chart, err := h.adminService.GetGrowthChart(r.Context(), days)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get growth chart", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: chart})
}
