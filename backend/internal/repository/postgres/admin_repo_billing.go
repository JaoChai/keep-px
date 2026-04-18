package postgres

import (
	"context"
	"fmt"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

func (r *AdminRepo) ListAllPurchases(ctx context.Context, status string, limit, offset int) ([]*domain.AdminPurchase, int, error) {
	var total int
	var countArgs []interface{}
	countQuery := `SELECT COUNT(*) FROM purchases`
	if status != "" {
		countQuery += ` WHERE status = $1`
		countArgs = append(countArgs, status)
	}
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	selectQuery := `SELECT p.id, p.customer_id, p.stripe_checkout_session_id, p.stripe_payment_intent_id,
	                       p.pack_type, p.amount_satang, p.currency, p.status, p.created_at, p.completed_at,
	                       c.email, c.name
	                FROM purchases p JOIN customers c ON c.id = p.customer_id`
	var args []interface{}
	argIdx := 1
	if status != "" {
		selectQuery += fmt.Sprintf(` WHERE p.status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}
	selectQuery += fmt.Sprintf(` ORDER BY p.created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var purchases []*domain.AdminPurchase
	for rows.Next() {
		ap := &domain.AdminPurchase{}
		if err := rows.Scan(
			&ap.ID, &ap.CustomerID, &ap.StripeCheckoutSessionID, &ap.StripePaymentIntentID,
			&ap.PackType, &ap.AmountSatang, &ap.Currency, &ap.Status, &ap.CreatedAt, &ap.CompletedAt,
			&ap.CustomerEmail, &ap.CustomerName,
		); err != nil {
			return nil, 0, err
		}
		purchases = append(purchases, ap)
	}
	if purchases == nil {
		purchases = []*domain.AdminPurchase{}
	}
	return purchases, total, rows.Err()
}

func (r *AdminRepo) ListAllSubscriptions(ctx context.Context, status string, limit, offset int) ([]*domain.AdminSubscription, int, error) {
	var total int
	var countArgs []interface{}
	countQuery := `SELECT COUNT(*) FROM subscriptions`
	if status != "" {
		countQuery += ` WHERE status = $1`
		countArgs = append(countArgs, status)
	}
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	selectQuery := `SELECT s.id, s.customer_id, s.stripe_subscription_id, s.stripe_price_id,
	                       s.addon_type, s.status, s.current_period_start, s.current_period_end,
	                       s.cancel_at_period_end, s.created_at, s.updated_at,
	                       c.email, c.name
	                FROM subscriptions s JOIN customers c ON c.id = s.customer_id`
	var args []interface{}
	argIdx := 1
	if status != "" {
		selectQuery += fmt.Sprintf(` WHERE s.status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}
	selectQuery += fmt.Sprintf(` ORDER BY s.created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var subs []*domain.AdminSubscription
	for rows.Next() {
		as := &domain.AdminSubscription{}
		if err := rows.Scan(
			&as.ID, &as.CustomerID, &as.StripeSubscriptionID, &as.StripePriceID,
			&as.AddonType, &as.Status, &as.CurrentPeriodStart, &as.CurrentPeriodEnd,
			&as.CancelAtPeriodEnd, &as.CreatedAt, &as.UpdatedAt,
			&as.CustomerEmail, &as.CustomerName,
		); err != nil {
			return nil, 0, err
		}
		subs = append(subs, as)
	}
	if subs == nil {
		subs = []*domain.AdminSubscription{}
	}
	return subs, total, rows.Err()
}

func (r *AdminRepo) ListCreditGrants(ctx context.Context, limit, offset int) ([]*domain.AdminCreditGrantWithCustomer, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_credit_grants`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT g.id, g.admin_id, g.customer_id, g.pack_type, g.total_replays, g.max_events_per_replay,
		        g.expires_at, g.reason, g.credit_id, g.created_at,
		        c.email, c.name
		 FROM admin_credit_grants g JOIN customers c ON c.id = g.customer_id
		 ORDER BY g.created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var grants []*domain.AdminCreditGrantWithCustomer
	for rows.Next() {
		g := &domain.AdminCreditGrantWithCustomer{}
		if err := rows.Scan(
			&g.ID, &g.AdminID, &g.CustomerID, &g.PackType, &g.TotalReplays, &g.MaxEventsPerReplay,
			&g.ExpiresAt, &g.Reason, &g.CreditID, &g.CreatedAt,
			&g.CustomerEmail, &g.CustomerName,
		); err != nil {
			return nil, 0, err
		}
		grants = append(grants, g)
	}
	if grants == nil {
		grants = []*domain.AdminCreditGrantWithCustomer{}
	}
	return grants, total, rows.Err()
}
