package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
	validate    *validator.Validate
	logger      *slog.Logger
}

func NewAuthHandler(authService *service.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validate:    newValidator(),
		logger:      logger,
	}
}

func (h *AuthHandler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	var input service.GoogleAuthInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, FormatValidationErrors(err))
		return
	}

	tokens, err := h.authService.GoogleAuth(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidGoogleToken) {
			ErrorJSON(w, http.StatusUnauthorized, "invalid google token")
			return
		}
		if errors.Is(err, service.ErrAccountSuspended) {
			ErrorJSON(w, http.StatusForbidden, "account suspended")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "google auth failed", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: tokens})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	customer, err := h.authService.GetCustomerByID(r.Context(), customerID)
	if err != nil {
		ErrorJSON(w, http.StatusUnauthorized, "invalid token")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: customer})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	if err := h.authService.Logout(r.Context(), customerID); err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "logout failed", err)
		return
	}
	JSON(w, http.StatusOK, APIResponse{Message: "logged out"})
}

func (h *AuthHandler) RegenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	customer, err := h.authService.RegenerateAPIKey(r.Context(), customerID)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to regenerate api key", err)
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: customer})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(body); err != nil {
		ErrorJSON(w, http.StatusBadRequest, FormatValidationErrors(err))
		return
	}

	tokens, err := h.authService.RefreshTokens(r.Context(), body.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			ErrorJSON(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}
		if errors.Is(err, service.ErrAccountSuspended) {
			ErrorJSON(w, http.StatusForbidden, "account suspended")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "token refresh failed", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: tokens})
}
