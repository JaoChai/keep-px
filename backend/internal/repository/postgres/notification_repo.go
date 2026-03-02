package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

var ErrNotificationNotFound = errors.New("notification not found")

type NotificationRepo struct {
	pool *pgxpool.Pool
}

func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{pool: pool}
}

func (r *NotificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO notifications (customer_id, type, title, body, metadata)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		n.CustomerID, n.Type, n.Title, n.Body, n.Metadata,
	).Scan(&n.ID, &n.CreatedAt)
}

func (r *NotificationRepo) ListByCustomerID(ctx context.Context, customerID string, limit int) ([]*domain.Notification, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, customer_id, type, title, body, metadata, is_read, created_at, read_at
		 FROM notifications WHERE customer_id = $1
		 ORDER BY created_at DESC LIMIT $2`,
		customerID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := make([]*domain.Notification, 0, limit)
	for rows.Next() {
		n := &domain.Notification{}
		if err := rows.Scan(&n.ID, &n.CustomerID, &n.Type, &n.Title, &n.Body, &n.Metadata, &n.IsRead, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *NotificationRepo) CountUnread(ctx context.Context, customerID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE customer_id = $1 AND is_read = FALSE`,
		customerID,
	).Scan(&count)
	return count, err
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id, customerID string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE notifications SET is_read = TRUE, read_at = NOW()
		 WHERE id = $1 AND customer_id = $2 AND is_read = FALSE`,
		id, customerID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, customerID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET is_read = TRUE, read_at = NOW()
		 WHERE customer_id = $1 AND is_read = FALSE`,
		customerID,
	)
	return err
}
