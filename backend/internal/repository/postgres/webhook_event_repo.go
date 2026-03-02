package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookEventRepo struct {
	pool *pgxpool.Pool
}

func NewWebhookEventRepo(pool *pgxpool.Pool) *WebhookEventRepo {
	return &WebhookEventRepo{pool: pool}
}

func (r *WebhookEventRepo) CreateIfNotExists(ctx context.Context, stripeEventID string, eventType string) (bool, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO stripe_webhook_events (stripe_event_id, event_type) VALUES ($1, $2) ON CONFLICT DO NOTHING RETURNING stripe_event_id`,
		stripeEventID, eventType,
	).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil // duplicate
		}
		return false, err
	}
	return true, nil
}

func (r *WebhookEventRepo) Delete(ctx context.Context, stripeEventID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM stripe_webhook_events WHERE stripe_event_id = $1`,
		stripeEventID,
	)
	return err
}
