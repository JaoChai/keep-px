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

type RuleHandler struct {
	ruleService *service.RuleService
	validate    *validator.Validate
}

func NewRuleHandler(ruleService *service.RuleService) *RuleHandler {
	return &RuleHandler{
		ruleService: ruleService,
		validate:    validator.New(),
	}
}

func (h *RuleHandler) ListByPixel(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pixelID := chi.URLParam(r, "pixelId")

	rules, err := h.ruleService.ListByPixelID(r.Context(), customerID, pixelID)
	if err != nil {
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusNotFound, "pixel not found")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to list rules")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: rules})
}

func (h *RuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pixelID := chi.URLParam(r, "pixelId")

	var input service.CreateRuleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	rule, err := h.ruleService.Create(r.Context(), customerID, pixelID, input)
	if err != nil {
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusNotFound, "pixel not found")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to create rule")
		return
	}
	JSON(w, http.StatusCreated, APIResponse{Data: rule})
}

func (h *RuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	ruleID := chi.URLParam(r, "id")

	var input service.UpdateRuleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rule, err := h.ruleService.Update(r.Context(), customerID, ruleID, input)
	if err != nil {
		if errors.Is(err, service.ErrRuleNotFound) {
			ErrorJSON(w, http.StatusNotFound, "rule not found")
			return
		}
		if errors.Is(err, service.ErrPixelNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "not authorized")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to update rule")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: rule})
}

func (h *RuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	ruleID := chi.URLParam(r, "id")

	err := h.ruleService.Delete(r.Context(), customerID, ruleID)
	if err != nil {
		if errors.Is(err, service.ErrRuleNotFound) {
			ErrorJSON(w, http.StatusNotFound, "rule not found")
			return
		}
		if errors.Is(err, service.ErrPixelNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "not authorized")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to delete rule")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Message: "rule deleted"})
}
