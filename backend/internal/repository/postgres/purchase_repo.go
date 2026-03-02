package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type PurchaseRepo struct {
	pool *pgxpool.Pool
}

func NewPurchaseRepo(pool *pgxpool.Pool) *PurchaseRepo {
	return &PurchaseRepo{pool: pool}
}

func scanPurchase(row pgx.Row) (*domain.Purchase, error) {
	p := &domain.Purchase{}
	err := row.Scan(
		&p.ID, &p.CustomerID, &p.StripeCheckoutSessionID, &p.StripePaymentIntentID,
		&p.PackType, &p.AmountSatang, &p.Currency, &p.Status,
		&p.CreatedAt, &p.CompletedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

const purchaseCols = `id, customer_id, stripe_checkout_session_id, stripe_payment_intent_id,
	pack_type, amount_satang, currency, status, created_at, completed_at`

func (r *PurchaseRepo) Create(ctx context.Context, p *domain.Purchase) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO purchases (customer_id, stripe_checkout_session_id, stripe_payment_intent_id,
			pack_type, amount_satang, currency, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at, completed_at`,
		p.CustomerID, p.StripeCheckoutSessionID, p.StripePaymentIntentID,
		p.PackType, p.AmountSatang, p.Currency, p.Status,
	).Scan(&p.ID, &p.CreatedAt, &p.CompletedAt)
}

func (r *PurchaseRepo) GetByID(ctx context.Context, id string) (*domain.Purchase, error) {
	return scanPurchase(r.pool.QueryRow(ctx,
		`SELECT `+purchaseCols+` FROM purchases WHERE id = $1`, id,
	))
}

func (r *PurchaseRepo) GetByStripeCheckoutSessionID(ctx context.Context, sessionID string) (*domain.Purchase, error) {
	return scanPurchase(r.pool.QueryRow(ctx,
		`SELECT `+purchaseCols+` FROM purchases WHERE stripe_checkout_session_id = $1`, sessionID,
	))
}

func (r *PurchaseRepo) UpdateStatus(ctx context.Context, id string, status string, completedAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE purchases SET status = $2, completed_at = $3 WHERE id = $1`,
		id, status, completedAt,
	)
	return err
}

func (r *PurchaseRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Purchase, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+purchaseCols+` FROM purchases WHERE customer_id = $1 ORDER BY created_at DESC`,
		customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var purchases []*domain.Purchase
	for rows.Next() {
		p := &domain.Purchase{}
		if err := rows.Scan(
			&p.ID, &p.CustomerID, &p.StripeCheckoutSessionID, &p.StripePaymentIntentID,
			&p.PackType, &p.AmountSatang, &p.Currency, &p.Status,
			&p.CreatedAt, &p.CompletedAt,
		); err != nil {
			return nil, err
		}
		purchases = append(purchases, p)
	}
	return purchases, rows.Err()
}
