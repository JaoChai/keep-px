package domain

// Plan constants — simplified to free vs paid
const (
	PlanSandbox = "sandbox"
	PlanPaid    = "paid"
)

// Free tier limits (no pixel slot subscription)
const (
	FreeMaxPixels         = 1
	FreeMaxSalePages      = 1
	FreeMaxEventsPerMonth = 5_000
	FreeRetentionDays     = 7
)

// Paid tier: per-slot values
const (
	PaidEventsPerSlot = 100_000
	PaidRetentionDays = 180
)

// DefaultMaxEventsPerReplay is used for all replay credits.
const DefaultMaxEventsPerReplay = 100_000

// CustomerQuota represents the resolved effective limits for a customer.
type CustomerQuota struct {
	PixelSlots          int    `json:"pixel_slots"`
	Plan                string `json:"plan"`
	MaxPixels           int    `json:"max_pixels"`
	MaxEventsPerMonth   int64  `json:"max_events_per_month"`
	EventsUsedThisMonth int64  `json:"events_used_this_month"`
	RetentionDays       int    `json:"retention_days"`
	MaxSalePages        int    `json:"max_sale_pages"`
	CanReplay           bool   `json:"can_replay"`
	RemainingReplays    int    `json:"remaining_replays"`
	MaxEventsPerReplay  int    `json:"max_events_per_replay"`
}
