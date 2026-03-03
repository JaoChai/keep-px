package domain

// Plan constants
const (
	PlanSandbox = "sandbox"
	PlanLaunch  = "launch"
	PlanShield  = "shield"
	PlanVault   = "vault"
)

// PlanLimits defines the base limits for a subscription plan.
type PlanLimits struct {
	MaxPixels         int
	MaxSalePages      int
	MaxEventsPerMonth int64
	RetentionDays     int
	IncludedReplays   int // -1 = unlimited, 0 = none (buy packs)
}

// PlanLimitsMap maps plan names to their base limits.
var PlanLimitsMap = map[string]PlanLimits{
	PlanSandbox: {MaxPixels: 2, MaxSalePages: 1, MaxEventsPerMonth: 5_000, RetentionDays: 7, IncludedReplays: 0},
	PlanLaunch:  {MaxPixels: 10, MaxSalePages: 5, MaxEventsPerMonth: 200_000, RetentionDays: 60, IncludedReplays: 0},
	PlanShield:  {MaxPixels: 25, MaxSalePages: 15, MaxEventsPerMonth: 1_000_000, RetentionDays: 180, IncludedReplays: 3},
	PlanVault:   {MaxPixels: 50, MaxSalePages: 30, MaxEventsPerMonth: 5_000_000, RetentionDays: 365, IncludedReplays: -1},
}

// Add-on values
const (
	Addon1MEventsPerMonth = 1_000_000
	AddonPixels10Extra    = 10
	AddonSalePages10Extra = 10
)

// DefaultMaxEventsPerReplay is used for all replay credits.
const DefaultMaxEventsPerReplay = 100_000

// CustomerQuota represents the resolved effective limits for a customer.
type CustomerQuota struct {
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
