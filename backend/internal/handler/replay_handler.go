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

type ReplayHandler struct {
	replayService *service.ReplayService
	validate      *validator.Validate
}

func NewReplayHandler(replayService *service.ReplayService) *ReplayHandler {
	return &ReplayHandler{
		replayService: replayService,
		validate:      validator.New(),
	}
}

func (h *ReplayHandler) Create(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	var input service.CreateReplayInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.replayService.Create(r.Context(), customerID, input)
	if err != nil {
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusNotFound, "pixel not found")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to create replay session")
		return
	}
	JSON(w, http.StatusCreated, APIResponse{Data: result.Session, Message: result.Warning})
}

func (h *ReplayHandler) List(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	sessions, err := h.replayService.List(r.Context(), customerID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to list replay sessions")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: sessions})
}

func (h *ReplayHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	sessionID := chi.URLParam(r, "id")

	session, err := h.replayService.GetByID(r.Context(), customerID, sessionID)
	if err != nil {
		if errors.Is(err, service.ErrReplayNotFound) {
			ErrorJSON(w, http.StatusNotFound, "replay session not found")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to get replay session")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: session})
}

func (h *ReplayHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	sessionID := chi.URLParam(r, "id")

	session, err := h.replayService.Cancel(r.Context(), customerID, sessionID)
	if err != nil {
		if errors.Is(err, service.ErrReplayNotFound) {
			ErrorJSON(w, http.StatusNotFound, "replay session not found")
			return
		}
		if errors.Is(err, service.ErrReplayNotCancellable) {
			ErrorJSON(w, http.StatusConflict, "replay session cannot be cancelled")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to cancel replay session")
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
		ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.replayService.Preview(r.Context(), customerID, input)
	if err != nil {
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusNotFound, "pixel not found")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to preview replay")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: result})
}

func (h *ReplayHandler) Retry(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	sessionID := chi.URLParam(r, "id")

	session, err := h.replayService.Retry(r.Context(), customerID, sessionID)
	if err != nil {
		if errors.Is(err, service.ErrReplayNotFound) {
			ErrorJSON(w, http.StatusNotFound, "replay session not found")
			return
		}
		if errors.Is(err, service.ErrReplayNotRetryable) {
			ErrorJSON(w, http.StatusConflict, "replay session cannot be retried")
			return
		}
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusNotFound, "pixel not found")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to retry replay session")
		return
	}
	JSON(w, http.StatusCreated, APIResponse{Data: session})
}
