package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

type CustomDomainHandler struct {
	service     *service.CustomDomainService
	validate    *validator.Validate
	cnameTarget string
	logger      *slog.Logger
}

func NewCustomDomainHandler(service *service.CustomDomainService, cnameTarget string, logger *slog.Logger) *CustomDomainHandler {
	return &CustomDomainHandler{
		service:     service,
		validate:    validator.New(),
		cnameTarget: cnameTarget,
		logger:      logger,
	}
}

// customDomainResponse is a response struct that excludes CFHostnameID.
type customDomainResponse struct {
	ID                string     `json:"id"`
	CustomerID        string     `json:"customer_id"`
	SalePageID        string     `json:"sale_page_id"`
	Domain            string     `json:"domain"`
	VerificationToken string     `json:"verification_token"`
	DNSVerified       bool       `json:"dns_verified"`
	SSLActive         bool       `json:"ssl_active"`
	VerifiedAt        *time.Time `json:"verified_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func toResponse(d *domain.CustomDomain) customDomainResponse {
	return customDomainResponse{
		ID:                d.ID,
		CustomerID:        d.CustomerID,
		SalePageID:        d.SalePageID,
		Domain:            d.Domain,
		VerificationToken: d.VerificationToken,
		DNSVerified:       d.DNSVerified,
		SSLActive:         d.SSLActive,
		VerifiedAt:        d.VerifiedAt,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}
}

func toResponseList(domains []*domain.CustomDomain) []customDomainResponse {
	result := make([]customDomainResponse, len(domains))
	for i, d := range domains {
		result[i] = toResponse(d)
	}
	return result
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

	d, err := h.service.Create(r.Context(), customerID, input)
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
		h.logger.Error("failed to create custom domain", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to create custom domain")
		return
	}

	JSON(w, http.StatusCreated, APIResponse{Data: map[string]any{
		"domain":       toResponse(d),
		"cname_target": h.cnameTarget,
	}})
}

func (h *CustomDomainHandler) List(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	domains, err := h.service.List(r.Context(), customerID)
	if err != nil {
		h.logger.Error("failed to list custom domains", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to list custom domains")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: toResponseList(domains)})
}

func (h *CustomDomainHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	d, err := h.service.GetByID(r.Context(), customerID, id)
	if err != nil {
		if errors.Is(err, service.ErrCustomDomainNotFound) {
			ErrorJSON(w, http.StatusNotFound, "custom domain not found")
			return
		}
		if errors.Is(err, service.ErrCustomDomainNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "custom domain not owned by you")
			return
		}
		h.logger.Error("failed to get custom domain", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to get custom domain")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: toResponse(d)})
}

func (h *CustomDomainHandler) Verify(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	d, err := h.service.Verify(r.Context(), customerID, id)
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
		h.logger.Error("failed to verify custom domain", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to verify custom domain")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: toResponse(d)})
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
		h.logger.Error("failed to delete custom domain", "error", err)
		ErrorJSON(w, http.StatusInternalServerError, "failed to delete custom domain")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Message: "custom domain deleted"})
}
