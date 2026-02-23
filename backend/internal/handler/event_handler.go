package handler

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

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

	created, err := h.eventService.Ingest(r.Context(), customerID, input)
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
