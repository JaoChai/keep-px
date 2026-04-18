package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

func (r *AdminRepo) ListCustomers(ctx context.Context, search, plan, status string, limit, offset int) ([]*domain.Customer, int, error) {
	baseWhere := baseWhereTrue
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (email ILIKE $%d OR name ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if plan != "" {
		baseWhere += fmt.Sprintf(" AND plan = $%d", argIdx)
		args = append(args, plan)
		argIdx++
	}
	switch status {
	case "suspended":
		baseWhere += " AND suspended_at IS NOT NULL"
	case "active":
		baseWhere += " AND suspended_at IS NULL"
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM customers " + baseWhere
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count customers: %w", err)
	}

	selectQuery := fmt.Sprintf(
		`SELECT id, email, password_hash, google_id, name, api_key, plan, stripe_customer_id, is_admin, suspended_at, created_at, updated_at
		 FROM customers %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		baseWhere, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	var customers []*domain.Customer
	for rows.Next() {
		c, err := scanCustomerFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		customers = append(customers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return customers, total, nil
}

func scanCustomerFromRows(rows pgx.Rows) (*domain.Customer, error) {
	c := &domain.Customer{}
	var passwordHash *string
	err := rows.Scan(
		&c.ID, &c.Email, &passwordHash, &c.GoogleID,
		&c.Name, &c.APIKey, &c.Plan, &c.StripeCustomerID,
		&c.IsAdmin, &c.SuspendedAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if passwordHash != nil {
		c.PasswordHash = *passwordHash
	}
	return c, nil
}

func (r *AdminRepo) GetCustomerDetail(ctx context.Context, id string) (*domain.AdminCustomerDetail, error) {
	customer, err := scanCustomer(r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, google_id, name, api_key, plan, stripe_customer_id, is_admin, suspended_at, created_at, updated_at
		 FROM customers WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil, nil
	}

	var (
		pixelCount    int
		eventCount    int64
		salePageCount int
		replayCount   int
		purchases     []*domain.Purchase
		credits       []*domain.ReplayCredit
		subscriptions []*domain.Subscription
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixels WHERE customer_id = $1`, id,
		).Scan(&pixelCount)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events pe JOIN pixels p ON p.id = pe.pixel_id WHERE p.customer_id = $1`, id,
		).Scan(&eventCount)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM sale_pages WHERE customer_id = $1`, id,
		).Scan(&salePageCount)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM replay_sessions WHERE customer_id = $1`, id,
		).Scan(&replayCount)
	})

	g.Go(func() error {
		rows, err := r.pool.Query(gCtx,
			`SELECT id, customer_id, stripe_checkout_session_id, stripe_payment_intent_id, pack_type, amount_satang, currency, status, created_at, completed_at
			 FROM purchases WHERE customer_id = $1 ORDER BY created_at DESC`, id,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			p := &domain.Purchase{}
			if err := rows.Scan(&p.ID, &p.CustomerID, &p.StripeCheckoutSessionID, &p.StripePaymentIntentID, &p.PackType, &p.AmountSatang, &p.Currency, &p.Status, &p.CreatedAt, &p.CompletedAt); err != nil {
				return err
			}
			purchases = append(purchases, p)
		}
		return rows.Err()
	})

	g.Go(func() error {
		rows, err := r.pool.Query(gCtx,
			`SELECT id, customer_id, purchase_id, pack_type, total_replays, used_replays, max_events_per_replay, expires_at, created_at
			 FROM replay_credits WHERE customer_id = $1 ORDER BY created_at DESC`, id,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			c := &domain.ReplayCredit{}
			if err := rows.Scan(&c.ID, &c.CustomerID, &c.PurchaseID, &c.PackType, &c.TotalReplays, &c.UsedReplays, &c.MaxEventsPerReplay, &c.ExpiresAt, &c.CreatedAt); err != nil {
				return err
			}
			credits = append(credits, c)
		}
		return rows.Err()
	})

	g.Go(func() error {
		rows, err := r.pool.Query(gCtx,
			`SELECT id, customer_id, stripe_subscription_id, stripe_price_id, addon_type, status, current_period_start, current_period_end, cancel_at_period_end, created_at, updated_at
			 FROM subscriptions WHERE customer_id = $1 ORDER BY created_at DESC`, id,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			s := &domain.Subscription{}
			if err := rows.Scan(&s.ID, &s.CustomerID, &s.StripeSubscriptionID, &s.StripePriceID, &s.AddonType, &s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelAtPeriodEnd, &s.CreatedAt, &s.UpdatedAt); err != nil {
				return err
			}
			subscriptions = append(subscriptions, s)
		}
		return rows.Err()
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get customer detail: %w", err)
	}

	if purchases == nil {
		purchases = []*domain.Purchase{}
	}
	if credits == nil {
		credits = []*domain.ReplayCredit{}
	}
	if subscriptions == nil {
		subscriptions = []*domain.Subscription{}
	}

	return &domain.AdminCustomerDetail{
		Customer:      customer,
		PixelCount:    pixelCount,
		EventCount:    eventCount,
		SalePageCount: salePageCount,
		ReplayCount:   replayCount,
		Purchases:     purchases,
		Credits:       credits,
		Subscriptions: subscriptions,
	}, nil
}

func (r *AdminRepo) SuspendCustomer(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE customers SET suspended_at = NOW(), updated_at = NOW() WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("suspend customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *AdminRepo) ActivateCustomer(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE customers SET suspended_at = NULL, updated_at = NOW() WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("activate customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}
