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

type PageStyle struct {
	BgColor     string `json:"bg_color,omitempty"`
	AccentColor string `json:"accent_color,omitempty"`
	TextColor   string `json:"text_color,omitempty"`
	BgImageURL  string `json:"bg_image_url,omitempty"`
}

type SimpleContent struct {
	Hero     HeroSection    `json:"hero"`
	Body     BodySection    `json:"body"`
	CTA      CTASection     `json:"cta"`
	Contact  ContactInfo    `json:"contact"`
	Tracking TrackingConfig `json:"tracking"`
	Style    PageStyle      `json:"style,omitempty"`
}

type TrackingConfig struct {
	CTAEventName string  `json:"cta_event_name"`
	ContentName  string  `json:"content_name"`
	ContentValue float64 `json:"content_value"`
	Currency     string  `json:"currency"`
}

type HeroSection struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	ImageURL string `json:"image_url"`
}

type BodySection struct {
	Description string   `json:"description"`
	Features    []string `json:"features"`
	Images      []string `json:"images"`
}

type CTASection struct {
	ButtonText string `json:"button_text"`
	ButtonLink string `json:"button_link"`
}

type ContactInfo struct {
	LineID     string `json:"line_id"`
	Phone      string `json:"phone"`
	WebsiteURL string `json:"website_url"`
}

// Block-based content (v2)

type BlockType string

const (
	BlockTypeImage  BlockType = "image"
	BlockTypeText   BlockType = "text"
	BlockTypeButton BlockType = "button"
)

type Block struct {
	ID          string    `json:"id"`
	Type        BlockType `json:"type"`
	ImageURL    string    `json:"image_url,omitempty"`
	LinkURL     string    `json:"link_url,omitempty"`
	Text        string    `json:"text,omitempty"`
	ButtonStyle string    `json:"button_style,omitempty"`
	ButtonText  string    `json:"button_text,omitempty"`
	ButtonURL   string    `json:"button_url,omitempty"`
	ButtonValue string    `json:"button_value,omitempty"`
}

type BlocksContent struct {
	Version  int            `json:"version"`
	Blocks   []Block        `json:"blocks"`
	Tracking TrackingConfig `json:"tracking"`
	Style    PageStyle      `json:"style,omitempty"`
}
