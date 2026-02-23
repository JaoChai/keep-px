package domain

import (
	"encoding/json"
	"time"
)

type EventRule struct {
	ID          string          `json:"id"`
	PixelID     string          `json:"pixel_id"`
	PageURL     string          `json:"page_url"`
	EventName   string          `json:"event_name"`
	TriggerType string          `json:"trigger_type"`
	CSSSelector string          `json:"css_selector,omitempty"`
	XPath       string          `json:"xpath,omitempty"`
	ElementText string          `json:"element_text,omitempty"`
	Conditions  json.RawMessage `json:"conditions,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	FireOnce    bool            `json:"fire_once"`
	DelayMs     int             `json:"delay_ms"`
	IsActive    bool            `json:"is_active"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
