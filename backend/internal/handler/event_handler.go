package handler

import (
	"encoding/json"
	"errors"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

type EventHandler struct {
	eventService *service.EventService
	validate     *validator.Validate
}

func NewEventHandler(eventService *service.EventService) *EventHandler {
	return &EventHandler{
		eventService: eventService,
		validate:     validator.New(),
	}
}

func (h *EventHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	var input service.IngestBatchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, FormatValidationErrors(err))
		return
	}

	clientIP := extractClientIP(r)
	clientUA := r.Header.Get("User-Agent")

	var fbc, fbp string
	if c, err := r.Cookie("_fbc"); err == nil {
		fbc = c.Value
	}
	if c, err := r.Cookie("_fbp"); err == nil {
		fbp = c.Value
	}

	created, err := h.eventService.Ingest(r.Context(), customerID, input, service.ClientContext{
		IP:        clientIP,
		UserAgent: clientUA,
		FBC:       fbc,
		FBP:       fbp,
	})
	if err != nil {
		if errors.Is(err, service.ErrQuotaEventsExceeded) {
			ErrorJSON(w, http.StatusPaymentRequired, "monthly event quota exceeded")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "ingestion failed")
		return
	}

	JSON(w, http.StatusAccepted, APIResponse{
		Data: map[string]int{"events_accepted": created},
	})
}

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	pixelID := r.URL.Query().Get("pixel_id")
	if pixelID != "" {
		if _, err := uuid.Parse(pixelID); err != nil {
			ErrorJSON(w, http.StatusBadRequest, "pixel_id must be a valid UUID")
			return
		}
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 50
	}

	events, total, err := h.eventService.ListByCustomerID(r.Context(), customerID, pixelID, page, perPage)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to list events")
		return
	}

	JSON(w, http.StatusOK, PaginatedResponse{
		Data:       events,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	})
}

func (h *EventHandler) ListRecent(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	pixelID := r.URL.Query().Get("pixel_id")
	if pixelID != "" {
		if _, err := uuid.Parse(pixelID); err != nil {
			ErrorJSON(w, http.StatusBadRequest, "pixel_id must be a valid UUID")
			return
		}
	}

	sinceStr := r.URL.Query().Get("since")

	// Initial load mode: no since parameter — service owns limit clamping
	if sinceStr == "" {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		events, err := h.eventService.ListLatest(r.Context(), customerID, pixelID, limit)
		if err != nil {
			ErrorJSON(w, http.StatusInternalServerError, "failed to fetch latest events")
			return
		}
		JSON(w, http.StatusOK, APIResponse{Data: events})
		return
	}

	// Polling mode: since parameter present
	since, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		ErrorJSON(w, http.StatusBadRequest, "since parameter must be in RFC3339 format")
		return
	}

	// Clamp since to max 5 minutes ago to prevent expensive historical scans
	maxLookback := time.Now().Add(-5 * time.Minute)
	if since.Before(maxLookback) {
		since = maxLookback
	}

	events, err := h.eventService.ListRecent(r.Context(), customerID, since, pixelID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to fetch recent events")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: events})
}

func (h *EventHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	eventID := chi.URLParam(r, "id")

	event, err := h.eventService.GetByID(r.Context(), customerID, eventID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to get event")
		return
	}
	if event == nil {
		ErrorJSON(w, http.StatusNotFound, "event not found")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: event})
}

// extractClientIP returns the real client IP by checking CDN/proxy headers first,
// then falling back to r.RemoteAddr. This improves Facebook EMQ score by providing
// the actual visitor IP instead of a load balancer or proxy address.
//
// Note: r.RemoteAddr may already be normalised by chimiddleware.RealIP (registered
// globally in router.go), which rewrites it from X-Real-IP / X-Forwarded-For.
func extractClientIP(r *http.Request) string {
	for _, h := range []string{"CF-Connecting-IP", "True-Client-IP"} {
		if raw := r.Header.Get(h); raw != "" {
			if ip := net.ParseIP(strings.TrimSpace(raw)); ip != nil {
				return ip.String()
			}
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
