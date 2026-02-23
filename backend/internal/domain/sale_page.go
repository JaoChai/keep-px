package domain

import (
	"encoding/json"
	"time"
)

type SalePage struct {
	ID           string          `json:"id"`
	CustomerID   string          `json:"customer_id"`
	PixelID      *string         `json:"pixel_id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	TemplateName string          `json:"template_name"`
	Content      json.RawMessage `json:"content"`
	IsPublished  bool            `json:"is_published"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type SimpleContent struct {
	Hero    HeroSection `json:"hero"`
	Body    BodySection `json:"body"`
	CTA     CTASection  `json:"cta"`
	Contact ContactInfo `json:"contact"`
}

type HeroSection struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	ImageURL string `json:"image_url"`
}

type BodySection struct {
	Description string   `json:"description"`
	Features    []string `json:"features"`
}

type CTASection struct {
	ButtonText string `json:"button_text"`
	ButtonLink string `json:"button_link"`
}

type ContactInfo struct {
	LineID string `json:"line_id"`
	Phone  string `json:"phone"`
}
