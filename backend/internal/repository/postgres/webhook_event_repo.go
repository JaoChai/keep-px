package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookEventRepo struct {
	pool *pgxpool.Pool
}

func NewWebhookEventRepo(pool *pgxpool.Pool) *WebhookEventRepo {
	return &WebhookEventRepo{pool: pool}
}

func (r *WebhookEventRepo) Exists(ctx context.Context, stripeEventID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM stripe_webhook_events WHERE stripe_event_id = $1)`,
		stripeEventID,
	).Scan(&exists)
	return exists, err
}

func (r *WebhookEventRepo) Create(ctx context.Context, stripeEventID, eventType string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO stripe_webhook_events (stripe_event_id, event_type) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		stripeEventID, eventType,
	)
	return err
}
