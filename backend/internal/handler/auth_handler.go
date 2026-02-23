package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
	validate    *validator.Validate
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validate:    validator.New(),
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input service.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	tokens, err := h.authService.Register(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			ErrorJSON(w, http.StatusConflict, "email already exists")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "registration failed")
		return
	}

	JSON(w, http.StatusCreated, APIResponse{Data: tokens})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input service.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	tokens, err := h.authService.Login(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			ErrorJSON(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "login failed")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: tokens})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.RefreshToken == "" {
		ErrorJSON(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	tokens, err := h.authService.RefreshTokens(r.Context(), body.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			ErrorJSON(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "token refresh failed")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: tokens})
}
