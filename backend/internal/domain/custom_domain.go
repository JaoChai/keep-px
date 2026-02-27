package domain

import "time"

type CustomDomain struct {
	ID                string     `json:"id"`
	CustomerID        string     `json:"customer_id"`
	SalePageID        string     `json:"sale_page_id"`
	Domain            string     `json:"domain"`
	CFHostnameID      *string    `json:"cf_hostname_id"`
	VerificationToken string     `json:"verification_token"`
	DNSVerified       bool       `json:"dns_verified"`
	SSLActive         bool       `json:"ssl_active"`
	VerifiedAt        *time.Time `json:"verified_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
