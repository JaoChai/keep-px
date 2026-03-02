package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type EventUsageRepo struct {
	pool *pgxpool.Pool
}

func NewEventUsageRepo(pool *pgxpool.Pool) *EventUsageRepo {
	return &EventUsageRepo{pool: pool}
}

func (r *EventUsageRepo) IncrementCount(ctx context.Context, customerID string, count int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO event_usage (customer_id, month, event_count)
		 VALUES ($1, date_trunc('month', CURRENT_DATE), $2)
		 ON CONFLICT (customer_id, month)
		 DO UPDATE SET event_count = event_usage.event_count + $2, updated_at = NOW()`,
		customerID, count,
	)
	return err
}

func (r *EventUsageRepo) GetCurrentMonth(ctx context.Context, customerID string) (*domain.EventUsage, error) {
	u := &domain.EventUsage{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, month, event_count, updated_at
		 FROM event_usage
		 WHERE customer_id = $1 AND month = date_trunc('month', CURRENT_DATE)`,
		customerID,
	).Scan(&u.ID, &u.CustomerID, &u.Month, &u.EventCount, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}
