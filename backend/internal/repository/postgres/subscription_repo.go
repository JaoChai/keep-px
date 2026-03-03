package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type SubscriptionRepo struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepo(pool *pgxpool.Pool) *SubscriptionRepo {
	return &SubscriptionRepo{pool: pool}
}

func scanSubscription(row pgx.Row) (*domain.Subscription, error) {
	s := &domain.Subscription{}
	err := row.Scan(
		&s.ID, &s.CustomerID, &s.StripeSubscriptionID, &s.StripePriceID,
		&s.AddonType, &s.Status,
		&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelAtPeriodEnd,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

const subscriptionCols = `id, customer_id, stripe_subscription_id, stripe_price_id,
	addon_type, status, current_period_start, current_period_end, cancel_at_period_end,
	created_at, updated_at`

func (r *SubscriptionRepo) Create(ctx context.Context, s *domain.Subscription) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO subscriptions (customer_id, stripe_subscription_id, stripe_price_id,
			addon_type, status, current_period_start, current_period_end, cancel_at_period_end)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at, updated_at`,
		s.CustomerID, s.StripeSubscriptionID, s.StripePriceID,
		s.AddonType, s.Status,
		s.CurrentPeriodStart, s.CurrentPeriodEnd, s.CancelAtPeriodEnd,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *SubscriptionRepo) GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*domain.Subscription, error) {
	return scanSubscription(r.pool.QueryRow(ctx,
		`SELECT `+subscriptionCols+` FROM subscriptions WHERE stripe_subscription_id = $1`, stripeSubID,
	))
}

func (r *SubscriptionRepo) GetActiveByCustomerID(ctx context.Context, customerID string) ([]*domain.Subscription, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+subscriptionCols+`
		 FROM subscriptions
		 WHERE customer_id = $1 AND status = 'active'
		 ORDER BY created_at DESC`,
		customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*domain.Subscription
	for rows.Next() {
		s := &domain.Subscription{}
		if err := rows.Scan(
			&s.ID, &s.CustomerID, &s.StripeSubscriptionID, &s.StripePriceID,
			&s.AddonType, &s.Status,
			&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelAtPeriodEnd,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (r *SubscriptionRepo) GetMaxEventsPerMonth(ctx context.Context, customerID string) (int64, error) {
	var plan string
	var addonBonus int64
	err := r.pool.QueryRow(ctx,
		`SELECT c.plan,
		  COALESCE((SELECT SUM(1000000) FROM subscriptions s
		     WHERE s.customer_id = $1 AND s.status = 'active' AND s.addon_type = 'events_1m'), 0)
		FROM customers c WHERE c.id = $1`,
		customerID,
	).Scan(&plan, &addonBonus)
	if err != nil {
		return 0, err
	}

	planLimits, ok := domain.PlanLimitsMap[plan]
	if !ok {
		planLimits = domain.PlanLimitsMap[domain.PlanSandbox]
	}

	return planLimits.MaxEventsPerMonth + addonBonus, nil
}

func (r *SubscriptionRepo) Update(ctx context.Context, s *domain.Subscription) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE subscriptions SET status = $2, current_period_start = $3, current_period_end = $4,
			cancel_at_period_end = $5, updated_at = NOW()
		 WHERE id = $1`,
		s.ID, s.Status, s.CurrentPeriodStart, s.CurrentPeriodEnd, s.CancelAtPeriodEnd,
	)
	return err
}

func (r *SubscriptionRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Subscription, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+subscriptionCols+`
		 FROM subscriptions
		 WHERE customer_id = $1
		 ORDER BY created_at DESC`,
		customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*domain.Subscription
	for rows.Next() {
		s := &domain.Subscription{}
		if err := rows.Scan(
			&s.ID, &s.CustomerID, &s.StripeSubscriptionID, &s.StripePriceID,
			&s.AddonType, &s.Status,
			&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelAtPeriodEnd,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}
