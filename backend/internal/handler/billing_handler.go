package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

type BillingHandler struct {
	billingService *service.BillingService
	quotaService   *service.QuotaService
	cfg            *config.Config
	logger         *slog.Logger
}

func NewBillingHandler(billingService *service.BillingService, quotaService *service.QuotaService, cfg *config.Config, logger *slog.Logger) *BillingHandler {
	return &BillingHandler{
		billingService: billingService,
		quotaService:   quotaService,
		cfg:            cfg,
		logger:         logger,
	}
}

func (h *BillingHandler) GetBillingOverview(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	overview, err := h.billingService.GetBillingOverview(r.Context(), customerID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to get billing overview")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: overview})
}

func (h *BillingHandler) GetQuota(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	quota, err := h.quotaService.GetCustomerQuota(r.Context(), customerID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to get quota")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: quota})
}

type createCheckoutInput struct {
	PackType  string `json:"pack_type,omitempty"`
	AddonType string `json:"addon_type,omitempty"`
}

func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	var input createCheckoutInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var checkoutURL string
	var err error

	switch {
	case input.PackType != "":
		checkoutURL, err = h.billingService.CreateReplayPackCheckout(r.Context(), customerID, input.PackType)
	case input.AddonType != "":
		checkoutURL, err = h.billingService.CreateAddonSubscriptionCheckout(r.Context(), customerID, input.AddonType)
	default:
		ErrorJSON(w, http.StatusBadRequest, "pack_type or addon_type is required")
		return
	}

	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidPackType):
			ErrorJSON(w, http.StatusBadRequest, "invalid pack type")
		case errors.Is(err, service.ErrInvalidAddonType):
			ErrorJSON(w, http.StatusBadRequest, "invalid addon type")
		case errors.Is(err, service.ErrStripeNotConfigured):
			ErrorJSON(w, http.StatusServiceUnavailable, "billing is not configured")
		case errors.Is(err, service.ErrCustomerNotFound):
			ErrorJSON(w, http.StatusNotFound, "customer not found")
		default:
			h.logger.Error("create checkout failed", "error", err, "customer_id", customerID)
			ErrorJSON(w, http.StatusInternalServerError, "failed to create checkout session")
		}
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: map[string]string{"url": checkoutURL}})
}

func (h *BillingHandler) CreatePortalSession(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	url, err := h.billingService.CreateCustomerPortalSession(r.Context(), customerID)
	if err != nil {
		if errors.Is(err, service.ErrStripeNotConfigured) {
			ErrorJSON(w, http.StatusServiceUnavailable, "billing is not configured")
			return
		}
		h.logger.Error("create portal session failed", "error", err, "customer_id", customerID)
		ErrorJSON(w, http.StatusInternalServerError, "failed to create portal session")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: map[string]string{"url": url}})
}

func (h *BillingHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = 65536
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		ErrorJSON(w, http.StatusBadRequest, "failed to read body")
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(body, sigHeader, h.cfg.StripeWebhookSecret)
	if err != nil {
		h.logger.Warn("webhook signature verification failed", "error", err)
		ErrorJSON(w, http.StatusBadRequest, "invalid signature")
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			h.logger.Error("failed to unmarshal checkout session", "error", err)
			ErrorJSON(w, http.StatusBadRequest, "invalid event data")
			return
		}
		if err := h.billingService.HandleCheckoutCompleted(r.Context(), &sess); err != nil {
			h.logger.Error("handle checkout completed failed", "error", err)
			ErrorJSON(w, http.StatusInternalServerError, "webhook processing failed")
			return
		}

	case "customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			h.logger.Error("failed to unmarshal subscription", "error", err)
			ErrorJSON(w, http.StatusBadRequest, "invalid event data")
			return
		}
		if err := h.billingService.HandleSubscriptionEvent(r.Context(), &sub, string(event.Type)); err != nil {
			h.logger.Error("handle subscription event failed", "error", err, "event_type", event.Type)
			ErrorJSON(w, http.StatusInternalServerError, "webhook processing failed")
			return
		}

	default:
		h.logger.Debug("unhandled webhook event type", "type", event.Type)
	}

	JSON(w, http.StatusOK, APIResponse{Message: "ok"})
}
