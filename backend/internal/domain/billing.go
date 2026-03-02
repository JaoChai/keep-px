package domain

import "time"

// Pack types
const (
	PackReplay1         = "replay_1"
	PackReplay3         = "replay_3"
	PackReplayUnlimited = "replay_unlimited"
)

// Add-on types
const (
	AddonRetention180 = "retention_180"
	AddonRetention365 = "retention_365"
	AddonEvents1M     = "events_1m"
)

// Purchase statuses
const (
	PurchaseStatusPending   = "pending"
	PurchaseStatusCompleted = "completed"
	PurchaseStatusFailed    = "failed"
)

// Subscription statuses
const (
	SubStatusActive   = "active"
	SubStatusCanceled = "canceled"
	SubStatusPastDue  = "past_due"
)

type Purchase struct {
	ID                      string     `json:"id"`
	CustomerID              string     `json:"customer_id"`
	StripeCheckoutSessionID *string    `json:"stripe_checkout_session_id,omitempty"`
	StripePaymentIntentID   *string    `json:"stripe_payment_intent_id,omitempty"`
	PackType                string     `json:"pack_type"`
	AmountSatang            int        `json:"amount_satang"`
	Currency                string     `json:"currency"`
	Status                  string     `json:"status"`
	CreatedAt               time.Time  `json:"created_at"`
	CompletedAt             *time.Time `json:"completed_at,omitempty"`
}

type ReplayCredit struct {
	ID                 string    `json:"id"`
	CustomerID         string    `json:"customer_id"`
	PurchaseID         *string   `json:"purchase_id,omitempty"`
	PackType           string    `json:"pack_type"`
	TotalReplays       int       `json:"total_replays"`
	UsedReplays        int       `json:"used_replays"`
	MaxEventsPerReplay int       `json:"max_events_per_replay"`
	ExpiresAt          time.Time `json:"expires_at"`
	CreatedAt          time.Time `json:"created_at"`
}

// RemainingReplays returns the number of replays remaining.
// Returns -1 if unlimited.
func (c *ReplayCredit) RemainingReplays() int {
	if c.TotalReplays == -1 {
		return -1
	}
	remaining := c.TotalReplays - c.UsedReplays
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsActive returns true if the credit has remaining replays and hasn't expired.
func (c *ReplayCredit) IsActive() bool {
	if time.Now().After(c.ExpiresAt) {
		return false
	}
	return c.TotalReplays == -1 || c.UsedReplays < c.TotalReplays
}

type Subscription struct {
	ID                   string     `json:"id"`
	CustomerID           string     `json:"customer_id"`
	StripeSubscriptionID string     `json:"stripe_subscription_id"`
	StripePriceID        string     `json:"stripe_price_id"`
	AddonType            string     `json:"addon_type"`
	Status               string     `json:"status"`
	CurrentPeriodStart   *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd    bool       `json:"cancel_at_period_end"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type EventUsage struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Month      time.Time `json:"month"`
	EventCount int64     `json:"event_count"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type StripeWebhookEvent struct {
	ID            string    `json:"id"`
	StripeEventID string    `json:"stripe_event_id"`
	EventType     string    `json:"event_type"`
	ProcessedAt   time.Time `json:"processed_at"`
}
