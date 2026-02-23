package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepo struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepo(pool *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{pool: pool}
}

func (r *RefreshTokenRepo) Create(ctx context.Context, customerID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (customer_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		customerID, tokenHash, expiresAt,
	)
	return err
}

func (r *RefreshTokenRepo) GetByTokenHash(ctx context.Context, tokenHash string) (string, time.Time, error) {
	var customerID string
	var expiresAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT customer_id, expires_at FROM refresh_tokens
		 WHERE token_hash = $1 AND expires_at > NOW()`,
		tokenHash,
	).Scan(&customerID, &expiresAt)
	if err == pgx.ErrNoRows {
		return "", time.Time{}, nil
	}
	return customerID, expiresAt, err
}

func (r *RefreshTokenRepo) DeleteByCustomerID(ctx context.Context, customerID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE customer_id = $1`, customerID,
	)
	return err
}

func (r *RefreshTokenRepo) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash,
	)
	return err
}
