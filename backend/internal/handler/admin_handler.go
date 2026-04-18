package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"

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
		validate:     newValidator(),
		logger:       logger,
	}
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

// queryPerPage returns a clamped per_page value (1-100, default 20).
func queryPerPage(r *http.Request) int {
	perPage := queryInt(r, "per_page", 20)
	if perPage > 100 {
		perPage = 100
	}
	return perPage
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
