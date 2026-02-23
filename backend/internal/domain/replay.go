package domain

import "time"

type ReplaySession struct {
	ID             string     `json:"id"`
	CustomerID     string     `json:"customer_id"`
	SourcePixelID  string     `json:"source_pixel_id"`
	TargetPixelID  string     `json:"target_pixel_id"`
	Status         string     `json:"status"`
	TotalEvents    int        `json:"total_events"`
	ReplayedEvents int        `json:"replayed_events"`
	FailedEvents   int        `json:"failed_events"`
	EventTypes     []string   `json:"event_types,omitempty"`
	DateFrom       *time.Time `json:"date_from,omitempty"`
	DateTo         *time.Time `json:"date_to,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
