package domain

import "time"

type Pixel struct {
	ID            string    `json:"id"`
	CustomerID    string    `json:"customer_id"`
	FBPixelID     string    `json:"fb_pixel_id"`
	FBAccessToken string    `json:"-"`
	Name          string    `json:"name"`
	IsActive      bool      `json:"is_active"`
	Status        string    `json:"status"`
	BackupPixelID *string   `json:"backup_pixel_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
