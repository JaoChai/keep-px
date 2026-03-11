package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type APIResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func ErrorJSON(w http.ResponseWriter, status int, message string) {
	JSON(w, status, APIResponse{Error: message})
}

func ErrorJSONWithLog(w http.ResponseWriter, r *http.Request, logger *slog.Logger, status int, userMsg string, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		status = http.StatusGatewayTimeout
		userMsg = "request timed out, please try again"
	case errors.Is(err, context.Canceled):
		status = http.StatusServiceUnavailable
		userMsg = "request cancelled"
	}
	logger.Error(userMsg,
		"error", err,
		"status", status,
		"method", r.Method,
		"path", r.URL.Path,
	)
	JSON(w, status, APIResponse{Error: userMsg})
}
