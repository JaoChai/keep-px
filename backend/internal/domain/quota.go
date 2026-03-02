package domain

// Free tier limits
const (
	FreeMaxPixels          = 10
	FreeMaxEventsPerMonth  = 200_000
	FreeRetentionDays      = 60
	FreeMaxSalePages       = 5
	FreeMaxEventsPerReplay = 100_000
)

// Add-on values
const (
	Addon1MEventsPerMonth = 1_000_000
	AddonRetention180Days = 180
	AddonRetention365Days = 365
)

// CustomerQuota represents the resolved effective limits for a customer.
type CustomerQuota struct {
	MaxPixels           int   `json:"max_pixels"`
	MaxEventsPerMonth   int64 `json:"max_events_per_month"`
	EventsUsedThisMonth int64 `json:"events_used_this_month"`
	RetentionDays       int   `json:"retention_days"`
	MaxSalePages        int   `json:"max_sale_pages"`
	CanReplay           bool  `json:"can_replay"`
	RemainingReplays    int   `json:"remaining_replays"`
	MaxEventsPerReplay  int   `json:"max_events_per_replay"`
}
