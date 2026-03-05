package domain

import (
	"encoding/json"
	"time"
)

type AdminCreditGrant struct {
	ID                 string    `json:"id"`
	AdminID            string    `json:"admin_id"`
	CustomerID         string    `json:"customer_id"`
	PackType           string    `json:"pack_type"`
	TotalReplays       int       `json:"total_replays"`
	MaxEventsPerReplay int       `json:"max_events_per_replay"`
	ExpiresAt          time.Time `json:"expires_at"`
	Reason             string    `json:"reason,omitempty"`
	CreditID           *string   `json:"credit_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

type PlatformStats struct {
	TotalCustomers      int            `json:"total_customers"`
	ActiveCustomers     int            `json:"active_customers"`
	SuspendedCustomers  int            `json:"suspended_customers"`
	TotalPixels         int            `json:"total_pixels"`
	EventsToday         int64          `json:"events_today"`
	EventsThisMonth     int64          `json:"events_this_month"`
	TotalReplays        int            `json:"total_replays"`
	SuccessfulReplays   int            `json:"successful_replays"`
	FailedReplays       int            `json:"failed_replays"`
	TotalRevenueTHB     float64        `json:"total_revenue_thb"`
	RevenueThisMonthTHB float64        `json:"revenue_this_month_thb"`
	CustomersByPlan     map[string]int `json:"customers_by_plan"`
}

type AdminCustomerDetail struct {
	Customer      *Customer       `json:"customer"`
	PixelCount    int             `json:"pixel_count"`
	EventCount    int64           `json:"event_count"`
	SalePageCount int             `json:"sale_page_count"`
	ReplayCount   int             `json:"replay_count"`
	Purchases     []*Purchase     `json:"purchases"`
	Credits       []*ReplayCredit `json:"credits"`
	Subscriptions []*Subscription `json:"subscriptions"`
}

type AdminPurchase struct {
	Purchase
	CustomerEmail string `json:"customer_email"`
	CustomerName  string `json:"customer_name"`
}

type AdminSubscription struct {
	Subscription
	CustomerEmail string `json:"customer_email"`
	CustomerName  string `json:"customer_name"`
}

type AdminCreditGrantWithCustomer struct {
	AdminCreditGrant
	CustomerEmail string `json:"customer_email"`
	CustomerName  string `json:"customer_name"`
}

type RevenueChartPoint struct {
	Date          string `json:"date"`
	AmountSatang  int64  `json:"amount_satang"`
	PurchaseCount int    `json:"purchase_count"`
}

type GrowthChartPoint struct {
	Date           string `json:"date"`
	NewCustomers   int    `json:"new_customers"`
	TotalCustomers int    `json:"total_customers"`
}

// F1: Admin Sale Pages
type AdminSalePage struct {
	SalePage
	CustomerEmail string `json:"customer_email"`
	CustomerName  string `json:"customer_name"`
	EventCount    int64  `json:"event_count"`
}

type AdminSalePageDetail struct {
	SalePage      *SalePage `json:"sale_page"`
	CustomerEmail string    `json:"customer_email"`
	CustomerName  string    `json:"customer_name"`
	LinkedPixels  []*Pixel  `json:"linked_pixels"`
	EventCount    int64     `json:"event_count"`
}

// F2: Admin Pixels
type AdminPixel struct {
	Pixel
	CustomerEmail string `json:"customer_email"`
	CustomerName  string `json:"customer_name"`
	EventCount    int64  `json:"event_count"`
	SalePageCount int    `json:"sale_page_count"`
}

type AdminPixelDetail struct {
	Pixel           *Pixel      `json:"pixel"`
	CustomerEmail   string      `json:"customer_email"`
	CustomerName    string      `json:"customer_name"`
	EventCount      int64       `json:"event_count"`
	LinkedSalePages []*SalePage `json:"linked_sale_pages"`
}

// F3: Admin Replay Sessions
type AdminReplaySession struct {
	ReplaySession
	CustomerEmail   string `json:"customer_email"`
	CustomerName    string `json:"customer_name"`
	SourcePixelName string `json:"source_pixel_name"`
	TargetPixelName string `json:"target_pixel_name"`
}

type AdminReplaySessionDetail struct {
	Session       *ReplaySession `json:"session"`
	CustomerEmail string         `json:"customer_email"`
	CustomerName  string         `json:"customer_name"`
	SourcePixel   *Pixel         `json:"source_pixel"`
	TargetPixel   *Pixel         `json:"target_pixel"`
}

// F4: Admin Events
type AdminEvent struct {
	PixelEvent
	PixelName     string `json:"pixel_name"`
	CustomerEmail string `json:"customer_email"`
	CustomerName  string `json:"customer_name"`
}

type AdminEventStats struct {
	TotalToday       int64                  `json:"total_today"`
	TotalThisHour    int64                  `json:"total_this_hour"`
	CAPISuccessRate  float64                `json:"capi_success_rate"`
	CAPIFailureCount int64                  `json:"capi_failure_count"`
	TopEventTypes    []EventTypeCount       `json:"top_event_types"`
	Timeseries       []EventTimeseriesPoint `json:"timeseries"`
}

type EventTimeseriesPoint struct {
	Timestamp   string `json:"timestamp"`
	EventCount  int64  `json:"event_count"`
	CAPISuccess int64  `json:"capi_success"`
	CAPIFailure int64  `json:"capi_failure"`
}

type EventTypeCount struct {
	EventName string `json:"event_name"`
	Count     int64  `json:"count"`
}

// F5: Audit Log
type AuditLogEntry struct {
	ID               string          `json:"id"`
	AdminID          string          `json:"admin_id"`
	AdminEmail       string          `json:"admin_email"`
	Action           string          `json:"action"`
	TargetType       string          `json:"target_type"`
	TargetID         string          `json:"target_id"`
	TargetCustomerID *string         `json:"target_customer_id,omitempty"`
	CustomerEmail    string          `json:"customer_email,omitempty"`
	Details          json.RawMessage `json:"details,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}
