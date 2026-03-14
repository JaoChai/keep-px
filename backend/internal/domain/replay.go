package domain

import (
	"encoding/json"
	"time"
)

// Replay session status constants.
const (
	ReplayStatusPending   = "pending"
	ReplayStatusRunning   = "running"
	ReplayStatusCompleted = "completed"
	ReplayStatusFailed    = "failed"
	ReplayStatusCancelled = "cancelled"
)

type ReplaySession struct {
	ID                string          `json:"id"`
	CustomerID        string          `json:"customer_id"`
	SourcePixelID     string          `json:"source_pixel_id"`
	TargetPixelID     string          `json:"target_pixel_id"`
	Status            string          `json:"status"`
	TotalEvents       int             `json:"total_events"`
	ReplayedEvents    int             `json:"replayed_events"`
	FailedEvents      int             `json:"failed_events"`
	EventTypes        []string        `json:"event_types,omitempty"`
	DateFrom          *time.Time      `json:"date_from,omitempty"`
	DateTo            *time.Time      `json:"date_to,omitempty"`
	TimeMode          string          `json:"time_mode"`
	BatchDelayMs      int             `json:"batch_delay_ms"`
	ErrorMessage      *string         `json:"error_message,omitempty"`
	StartedAt         *time.Time      `json:"started_at,omitempty"`
	CompletedAt       *time.Time      `json:"completed_at,omitempty"`
	CancelledAt       *time.Time      `json:"cancelled_at,omitempty"`
	FailedBatchRanges json.RawMessage `json:"failed_batch_ranges,omitempty"`
	CreditID          *string         `json:"credit_id,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
}

// BatchRange represents a range of events that failed during replay.
type BatchRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}
