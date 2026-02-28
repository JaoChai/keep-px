package handler

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
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
		ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	clientIP := r.RemoteAddr
	clientUA := r.Header.Get("User-Agent")

	created, err := h.eventService.Ingest(r.Context(), customerID, input, clientIP, clientUA)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "ingestion failed")
		return
	}

	JSON(w, http.StatusAccepted, APIResponse{
		Data: map[string]int{"events_accepted": created},
	})
}

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 50
	}

	events, total, err := h.eventService.ListByCustomerID(r.Context(), customerID, page, perPage)
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

	sinceStr := r.URL.Query().Get("since")
	if sinceStr == "" {
		ErrorJSON(w, http.StatusBadRequest, "since parameter is required (RFC3339 format)")
		return
	}
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

	pixelID := r.URL.Query().Get("pixel_id")
	if pixelID != "" {
		if _, err := uuid.Parse(pixelID); err != nil {
			ErrorJSON(w, http.StatusBadRequest, "pixel_id must be a valid UUID")
			return
		}
	}

	events, err := h.eventService.ListRecent(r.Context(), customerID, since, pixelID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to fetch recent events")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: events})
}

func (h *EventHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")

	event, err := h.eventService.GetByID(r.Context(), eventID)
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
