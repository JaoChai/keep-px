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
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get billing overview", err)
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: overview})
}

func (h *BillingHandler) GetQuota(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	quota, err := h.quotaService.GetCustomerQuota(r.Context(), customerID)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get quota", err)
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: quota})
}

type createCheckoutInput struct {
	Type     string `json:"type" validate:"required,oneof=pixel_slots replay_single replay_monthly"`
	Quantity int    `json:"quantity,omitempty"`
}

func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	var input createCheckoutInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Type == "" {
		ErrorJSON(w, http.StatusBadRequest, "type is required")
		return
	}

	var checkoutURL string
	var err error

	switch input.Type {
	case "pixel_slots":
		if input.Quantity < 1 {
			input.Quantity = 1
		}
		checkoutURL, err = h.billingService.CreatePixelSlotCheckout(r.Context(), customerID, input.Quantity)
	case "replay_single", "replay_monthly":
		checkoutURL, err = h.billingService.CreateReplayCheckout(r.Context(), customerID, input.Type)
	default:
		ErrorJSON(w, http.StatusBadRequest, "invalid type: must be pixel_slots, replay_single, or replay_monthly")
		return
	}

	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCheckoutType):
			ErrorJSON(w, http.StatusBadRequest, "invalid checkout type")
		case errors.Is(err, service.ErrStripeNotConfigured):
			ErrorJSON(w, http.StatusServiceUnavailable, "billing is not configured")
		case errors.Is(err, service.ErrCustomerNotFound):
			ErrorJSON(w, http.StatusNotFound, "customer not found")
		default:
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to create checkout session", err)
		}
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: map[string]string{"url": checkoutURL}})
}

type updateSlotsInput struct {
	Quantity int `json:"quantity" validate:"required,min=1"`
}

func (h *BillingHandler) UpdateSlots(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	var input updateSlotsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Quantity < 1 {
		ErrorJSON(w, http.StatusBadRequest, "quantity must be at least 1")
		return
	}

	url, err := h.billingService.UpdatePixelSlotQuantity(r.Context(), customerID, input.Quantity)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStripeNotConfigured):
			ErrorJSON(w, http.StatusServiceUnavailable, "billing is not configured")
		case errors.Is(err, service.ErrCustomerNotFound):
			ErrorJSON(w, http.StatusNotFound, "customer not found")
		default:
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to update slots", err)
		}
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: map[string]string{"url": url}})
}

func (h *BillingHandler) CreatePortalSession(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	url, err := h.billingService.CreateCustomerPortalSession(r.Context(), customerID)
	if err != nil {
		if errors.Is(err, service.ErrStripeNotConfigured) {
			ErrorJSON(w, http.StatusServiceUnavailable, "billing is not configured")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to create portal session", err)
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
	event, err := webhook.ConstructEventWithOptions(body, sigHeader, h.cfg.StripeWebhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		h.logger.Warn("webhook signature verification failed", "error", err)
		ErrorJSON(w, http.StatusBadRequest, "invalid signature")
		return
	}

	var processErr error

	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			h.logger.Error("failed to unmarshal checkout session", "error", err)
			ErrorJSON(w, http.StatusBadRequest, "invalid event data")
			return
		}
		processErr = h.billingService.ProcessWebhookEvent(r.Context(), event.ID, string(event.Type), func() error {
			return h.billingService.HandleCheckoutCompleted(r.Context(), &sess)
		})

	case "customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			h.logger.Error("failed to unmarshal subscription", "error", err)
			ErrorJSON(w, http.StatusBadRequest, "invalid event data")
			return
		}
		processErr = h.billingService.ProcessWebhookEvent(r.Context(), event.ID, string(event.Type), func() error {
			return h.billingService.HandleSubscriptionEvent(r.Context(), &sub, string(event.Type))
		})

	default:
		h.logger.Debug("unhandled webhook event type", "type", event.Type)
	}

	if processErr != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "webhook processing failed", processErr)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Message: "ok"})
}
