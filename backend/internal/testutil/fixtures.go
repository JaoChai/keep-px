//go:build integration

package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// CreateTestCustomer inserts a customer with random email + API key and
// returns the created row (with ID / timestamps populated).
func CreateTestCustomer(t *testing.T, pool *pgxpool.Pool) *domain.Customer {
	t.Helper()
	c := &domain.Customer{
		Email:         fmt.Sprintf("test-%s@keep-px.local", randomHex(8)),
		Name:          "Test User",
		APIKey:        "pk_" + randomHex(16),
		Plan:          "free",
		RetentionDays: 7,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pool.QueryRow(ctx,
		`INSERT INTO customers (email, password_hash, name, api_key, plan, retention_days)
		 VALUES ($1, '', $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		c.Email, c.Name, c.APIKey, c.Plan, c.RetentionDays,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		t.Fatalf("create test customer: %v", err)
	}
	return c
}

// CreateTestPixel inserts a pixel for the given customer with a fixed
// fb_access_token "test-token" (plaintext — encryption is opt-in per repo).
func CreateTestPixel(t *testing.T, pool *pgxpool.Pool, customerID string) *domain.Pixel {
	t.Helper()
	p := &domain.Pixel{
		CustomerID:    customerID,
		FBPixelID:     fmt.Sprintf("123456789%d", time.Now().UnixNano()%10000),
		FBAccessToken: "test-token",
		Name:          "Test Pixel",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pool.QueryRow(ctx,
		`INSERT INTO pixels (customer_id, fb_pixel_id, fb_access_token, name)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, is_active, created_at, updated_at`,
		p.CustomerID, p.FBPixelID, p.FBAccessToken, p.Name,
	).Scan(&p.ID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		t.Fatalf("create test pixel: %v", err)
	}
	return p
}

// CreateTestEvent inserts a pixel_event row and returns it. Caller may pass
// an empty PixelEvent — required fields are filled with sensible defaults.
func CreateTestEvent(t *testing.T, pool *pgxpool.Pool, pixelID string, overrides ...func(*domain.PixelEvent)) *domain.PixelEvent {
	t.Helper()
	e := &domain.PixelEvent{
		PixelID:   pixelID,
		EventName: "Purchase",
		EventData: json.RawMessage(`{"value":100,"currency":"THB"}`),
		UserData:  json.RawMessage(`{"em":["hashed-email"]}`),
		SourceURL: "https://example.com/checkout",
		EventTime: time.Now(),
	}
	for _, fn := range overrides {
		fn(e)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pool.QueryRow(ctx,
		`INSERT INTO pixel_events (pixel_id, event_name, event_data, user_data, source_url, event_time, event_id, client_ip, client_user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (pixel_id, event_id) WHERE event_id IS NOT NULL DO NOTHING
		 RETURNING id, forwarded_to_capi, created_at`,
		e.PixelID, e.EventName, e.EventData, e.UserData, e.SourceURL, e.EventTime, nullableString(e.EventID), nullableString(e.ClientIP), nullableString(e.ClientUserAgent),
	).Scan(&e.ID, &e.ForwardedToCAPI, &e.CreatedAt)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}
	return e
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
