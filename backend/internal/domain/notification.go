package domain

import (
	"encoding/json"
	"time"
)

const (
	NotificationTypeReplayCompleted = "replay_completed"
	NotificationTypeReplayFailed    = "replay_failed"
	NotificationTypeCAPIAuthError   = "capi_auth_error"
	NotificationTypeSystem          = "system"
)

type Notification struct {
	ID         string          `json:"id"`
	CustomerID string          `json:"customer_id"`
	Type       string          `json:"type"`
	Title      string          `json:"title"`
	Body       string          `json:"body"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	IsRead     bool            `json:"is_read"`
	CreatedAt  time.Time       `json:"created_at"`
	ReadAt     *time.Time      `json:"read_at,omitempty"`
}
