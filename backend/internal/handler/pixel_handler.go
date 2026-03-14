package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/jaochai/pixlinks/backend/internal/facebook"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

type PixelHandler struct {
	pixelService *service.PixelService
	validate     *validator.Validate
	logger       *slog.Logger
}

func NewPixelHandler(pixelService *service.PixelService, logger *slog.Logger) *PixelHandler {
	return &PixelHandler{
		pixelService: pixelService,
		validate:     validator.New(),
		logger:       logger,
	}
}

// mapPixelError maps pixel service sentinel errors to HTTP responses.
// Returns true if the error was handled, false otherwise.
func mapPixelError(err error, w http.ResponseWriter) bool {
	switch {
	case errors.Is(err, service.ErrPixelNotFound):
		ErrorJSON(w, http.StatusNotFound, "pixel not found")
	case errors.Is(err, service.ErrPixelNotOwned):
		ErrorJSON(w, http.StatusForbidden, "pixel not owned by you")
	case errors.Is(err, service.ErrInvalidFBPixelID):
		ErrorJSON(w, http.StatusBadRequest, "invalid Facebook Pixel ID: must be 15-16 digits")
	case errors.Is(err, service.ErrBackupPixelSelf):
		ErrorJSON(w, http.StatusBadRequest, "cannot set pixel as its own backup")
	case errors.Is(err, service.ErrBackupPixelNotFound):
		ErrorJSON(w, http.StatusBadRequest, "backup pixel not found")
	case errors.Is(err, service.ErrBackupPixelNotOwned):
		ErrorJSON(w, http.StatusForbidden, "backup pixel not owned by you")
	case errors.Is(err, service.ErrPixelNoAccessToken):
		ErrorJSON(w, http.StatusBadRequest, "pixel has no access token configured")
	case errors.Is(err, service.ErrQuotaPixelsExceeded):
		ErrorJSON(w, http.StatusPaymentRequired, "pixel limit exceeded")
	default:
		return false
	}
	return true
}

func (h *PixelHandler) Get(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pixelID := chi.URLParam(r, "id")

	pixel, err := h.pixelService.GetByID(r.Context(), customerID, pixelID)
	if err != nil {
		if !mapPixelError(err, w) {
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get pixel", err)
		}
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: pixel})
}

func (h *PixelHandler) List(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pixels, err := h.pixelService.List(r.Context(), customerID)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to list pixels", err)
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: pixels})
}

func (h *PixelHandler) Create(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	var input service.CreatePixelInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, FormatValidationErrors(err))
		return
	}

	pixel, err := h.pixelService.Create(r.Context(), customerID, input)
	if err != nil {
		if !mapPixelError(err, w) {
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to create pixel", err)
		}
		return
	}
	JSON(w, http.StatusCreated, APIResponse{Data: pixel})
}

func (h *PixelHandler) Update(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pixelID := chi.URLParam(r, "id")

	var input service.UpdatePixelInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pixel, err := h.pixelService.Update(r.Context(), customerID, pixelID, input)
	if err != nil {
		if !mapPixelError(err, w) {
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to update pixel", err)
		}
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: pixel})
}

func (h *PixelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pixelID := chi.URLParam(r, "id")

	err := h.pixelService.Delete(r.Context(), customerID, pixelID)
	if err != nil {
		if !mapPixelError(err, w) {
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to delete pixel", err)
		}
		return
	}
	JSON(w, http.StatusOK, APIResponse{Message: "pixel deleted"})
}

func (h *PixelHandler) Test(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pixelID := chi.URLParam(r, "id")

	resp, err := h.pixelService.TestConnection(r.Context(), customerID, pixelID)
	if err != nil {
		if mapPixelError(err, w) {
			return
		}
		// Check for Facebook CAPI error — never expose raw FB response to client
		var capiErr *facebook.CAPIError
		if errors.As(err, &capiErr) {
			h.logger.Error("pixel test connection CAPI error",
				"pixel_id", pixelID,
				"fb_status", capiErr.StatusCode,
				"fb_response", capiErr.Message,
			)
			ErrorJSON(w, http.StatusBadGateway, sanitizeCAPIError(capiErr.StatusCode))
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusBadGateway, "failed to test pixel connection", err)
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: resp})
}

func sanitizeCAPIError(statusCode int) string {
	switch statusCode {
	case 400:
		return "Facebook rejected the request. Check your Pixel ID and Access Token."
	case 401, 403:
		return "Facebook authentication failed. Your access token may be invalid or expired."
	case 429:
		return "Facebook rate limit exceeded. Please try again later."
	default:
		return fmt.Sprintf("Facebook returned an error (HTTP %d). Please try again.", statusCode)
	}
}
