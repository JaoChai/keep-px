package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

type ReplayHandler struct {
	replayService *service.ReplayService
	validate      *validator.Validate
	logger        *slog.Logger
}

func NewReplayHandler(replayService *service.ReplayService, logger *slog.Logger) *ReplayHandler {
	return &ReplayHandler{
		replayService: replayService,
		validate:      validator.New(),
		logger:        logger,
	}
}

// mapReplayError maps common replay service errors to HTTP responses.
// Returns true if an error was handled.
func mapReplayError(err error, w http.ResponseWriter) bool {
	switch {
	case errors.Is(err, service.ErrPixelNotFound):
		ErrorJSON(w, http.StatusNotFound, "pixel not found")
	case errors.Is(err, service.ErrReplaySamePixel):
		ErrorJSON(w, http.StatusBadRequest, "source and target pixel cannot be the same")
	case errors.Is(err, service.ErrPixelNoCredentials):
		ErrorJSON(w, http.StatusUnprocessableEntity, "target pixel has no Facebook credentials configured")
	case errors.Is(err, service.ErrReplayNotFound):
		ErrorJSON(w, http.StatusNotFound, "replay session not found")
	case errors.Is(err, service.ErrReplayNotCancellable):
		ErrorJSON(w, http.StatusConflict, "replay session cannot be cancelled")
	case errors.Is(err, service.ErrReplayNotRetryable):
		ErrorJSON(w, http.StatusConflict, "replay session cannot be retried")
	case errors.Is(err, service.ErrQuotaReplayNotAllowed):
		ErrorJSON(w, http.StatusPaymentRequired, "no replay credits available")
	case errors.Is(err, service.ErrQuotaReplayEventsExceeded):
		ErrorJSON(w, http.StatusPaymentRequired, "replay event count exceeds credit limit")
	default:
		return false
	}
	return true
}

func (h *ReplayHandler) Create(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	var input service.CreateReplayInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, FormatValidationErrors(err))
		return
	}

	result, err := h.replayService.Create(r.Context(), customerID, input)
	if err != nil {
		if !mapReplayError(err, w) {
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to create replay session", err)
		}
		return
	}
	JSON(w, http.StatusCreated, APIResponse{Data: result.Session, Message: result.Warning})
}

func (h *ReplayHandler) List(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	sessions, err := h.replayService.List(r.Context(), customerID)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to list replay sessions", err)
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: sessions})
}

func (h *ReplayHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	sessionID := chi.URLParam(r, "id")

	session, err := h.replayService.GetByID(r.Context(), customerID, sessionID)
	if err != nil {
		if !mapReplayError(err, w) {
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get replay session", err)
		}
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: session})
}

func (h *ReplayHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	sessionID := chi.URLParam(r, "id")

	session, err := h.replayService.Cancel(r.Context(), customerID, sessionID)
	if err != nil {
		if !mapReplayError(err, w) {
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to cancel replay session", err)
		}
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: session})
}

func (h *ReplayHandler) Preview(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	var input service.CreateReplayInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, FormatValidationErrors(err))
		return
	}

	result, err := h.replayService.Preview(r.Context(), customerID, input)
	if err != nil {
		if !mapReplayError(err, w) {
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to preview replay", err)
		}
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: result})
}

func (h *ReplayHandler) Retry(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	sessionID := chi.URLParam(r, "id")

	session, err := h.replayService.Retry(r.Context(), customerID, sessionID)
	if err != nil {
		if !mapReplayError(err, w) {
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to retry replay session", err)
		}
		return
	}
	JSON(w, http.StatusCreated, APIResponse{Data: session})
}

func (h *ReplayHandler) EventTypes(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pixelID := r.URL.Query().Get("pixel_id")
	if pixelID == "" {
		ErrorJSON(w, http.StatusBadRequest, "pixel_id is required")
		return
	}
	types, err := h.replayService.GetEventTypes(r.Context(), customerID, pixelID)
	if err != nil {
		if !mapReplayError(err, w) {
			ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to get event types", err)
		}
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: types})
}
