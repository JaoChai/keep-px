package domain

import "time"

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
