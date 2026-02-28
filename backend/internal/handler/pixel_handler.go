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
}

func NewPixelHandler(pixelService *service.PixelService) *PixelHandler {
	return &PixelHandler{
		pixelService: pixelService,
		validate:     validator.New(),
	}
}

func (h *PixelHandler) List(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pixels, err := h.pixelService.List(r.Context(), customerID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to list pixels")
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
		ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	pixel, err := h.pixelService.Create(r.Context(), customerID, input)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to create pixel")
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
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusNotFound, "pixel not found")
			return
		}
		if errors.Is(err, service.ErrPixelNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "pixel not owned by you")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to update pixel")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: pixel})
}

func (h *PixelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pixelID := chi.URLParam(r, "id")

	err := h.pixelService.Delete(r.Context(), customerID, pixelID)
	if err != nil {
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusNotFound, "pixel not found")
			return
		}
		if errors.Is(err, service.ErrPixelNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "pixel not owned by you")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to delete pixel")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Message: "pixel deleted"})
}

func (h *PixelHandler) Test(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pixelID := chi.URLParam(r, "id")

	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	userAgent := r.UserAgent()

	resp, err := h.pixelService.TestConnection(r.Context(), customerID, pixelID, clientIP, userAgent)
	if err != nil {
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusNotFound, "pixel not found")
			return
		}
		if errors.Is(err, service.ErrPixelNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "pixel not owned by you")
			return
		}
		if errors.Is(err, service.ErrPixelNoAccessToken) {
			ErrorJSON(w, http.StatusBadRequest, "pixel has no access token configured")
			return
		}
		// Check for Facebook CAPI error — never expose raw FB response to client
		var capiErr *facebook.CAPIError
		if errors.As(err, &capiErr) {
			slog.Error("pixel test connection CAPI error",
				"pixel_id", pixelID,
				"fb_status", capiErr.StatusCode,
				"fb_response", capiErr.Message,
			)
			ErrorJSON(w, http.StatusBadGateway, sanitizeCAPIError(capiErr.StatusCode))
			return
		}
		ErrorJSON(w, http.StatusBadGateway, "failed to test pixel connection")
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
