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

type CustomDomainHandler struct {
	service     *service.CustomDomainService
	validate    *validator.Validate
	cnameTarget string
}

func NewCustomDomainHandler(service *service.CustomDomainService, cnameTarget string) *CustomDomainHandler {
	return &CustomDomainHandler{
		service:     service,
		validate:    validator.New(),
		cnameTarget: cnameTarget,
	}
}

type createDomainResponse struct {
	Data        interface{} `json:"data"`
	CNAMETarget string      `json:"cname_target"`
}

func (h *CustomDomainHandler) Create(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	var input service.CreateCustomDomainInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	domain, err := h.service.Create(r.Context(), customerID, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidDomain) {
			ErrorJSON(w, http.StatusBadRequest, "invalid domain name")
			return
		}
		if errors.Is(err, service.ErrSalePageNotFound) {
			ErrorJSON(w, http.StatusNotFound, "sale page not found")
			return
		}
		if errors.Is(err, service.ErrSalePageNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "sale page not owned by you")
			return
		}
		if errors.Is(err, service.ErrSalePageNotPublished) {
			ErrorJSON(w, http.StatusBadRequest, "sale page must be published before adding a custom domain")
			return
		}
		if errors.Is(err, service.ErrDomainAlreadyExists) {
			ErrorJSON(w, http.StatusConflict, "domain already exists")
			return
		}
		if errors.Is(err, service.ErrCloudflareNotConfigured) {
			ErrorJSON(w, http.StatusServiceUnavailable, "cloudflare integration not configured")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to create custom domain")
		return
	}

	JSON(w, http.StatusCreated, createDomainResponse{
		Data:        domain,
		CNAMETarget: h.cnameTarget,
	})
}

func (h *CustomDomainHandler) List(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	domains, err := h.service.List(r.Context(), customerID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to list custom domains")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: domains})
}

func (h *CustomDomainHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	domain, err := h.service.GetByID(r.Context(), customerID, id)
	if err != nil {
		if errors.Is(err, service.ErrCustomDomainNotFound) {
			ErrorJSON(w, http.StatusNotFound, "custom domain not found")
			return
		}
		if errors.Is(err, service.ErrCustomDomainNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "custom domain not owned by you")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to get custom domain")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: domain})
}

func (h *CustomDomainHandler) Verify(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	domain, err := h.service.Verify(r.Context(), customerID, id)
	if err != nil {
		if errors.Is(err, service.ErrCustomDomainNotFound) {
			ErrorJSON(w, http.StatusNotFound, "custom domain not found")
			return
		}
		if errors.Is(err, service.ErrCustomDomainNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "custom domain not owned by you")
			return
		}
		if errors.Is(err, service.ErrCloudflareNotConfigured) {
			ErrorJSON(w, http.StatusServiceUnavailable, "cloudflare integration not configured")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to verify custom domain")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: domain})
}

func (h *CustomDomainHandler) Delete(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	err := h.service.Delete(r.Context(), customerID, id)
	if err != nil {
		if errors.Is(err, service.ErrCustomDomainNotFound) {
			ErrorJSON(w, http.StatusNotFound, "custom domain not found")
			return
		}
		if errors.Is(err, service.ErrCustomDomainNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "custom domain not owned by you")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to delete custom domain")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Message: "custom domain deleted"})
}
