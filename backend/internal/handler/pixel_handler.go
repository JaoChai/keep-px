package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

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
