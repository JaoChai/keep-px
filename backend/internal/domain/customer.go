package domain

import "time"

type Customer struct {
	ID               string     `json:"id"`
	Email            string     `json:"email"`
	PasswordHash     string     `json:"-"`
	GoogleID         *string    `json:"-"`
	Name             string     `json:"name"`
	APIKey           string     `json:"api_key"`
	Plan             string     `json:"plan"`
	RetentionDays    int        `json:"retention_days"`
	StripeCustomerID *string    `json:"stripe_customer_id,omitempty"`
	IsAdmin          bool       `json:"is_admin"`
	SuspendedAt      *time.Time `json:"suspended_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
