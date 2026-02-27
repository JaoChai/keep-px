package domain

import (
	"encoding/json"
	"time"
)

type PixelEvent struct {
	ID               string          `json:"id"`
	PixelID          string          `json:"pixel_id"`
	EventName        string          `json:"event_name"`
	EventData        json.RawMessage `json:"event_data"`
	UserData         json.RawMessage `json:"user_data,omitempty"`
	SourceURL        string          `json:"source_url,omitempty"`
	EventID          string          `json:"event_id,omitempty"`
	ClientIP         string          `json:"client_ip,omitempty"`
	ClientUserAgent  string          `json:"client_user_agent,omitempty"`
	EventTime        time.Time       `json:"event_time"`
	ForwardedToCAPI  bool            `json:"forwarded_to_capi"`
	CAPIResponseCode *int            `json:"capi_response_code,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}
