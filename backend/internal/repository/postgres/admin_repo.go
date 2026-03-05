package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

type AdminRepo struct {
	pool *pgxpool.Pool
}

func NewAdminRepo(pool *pgxpool.Pool) *AdminRepo {
	return &AdminRepo{pool: pool}
}

func (r *AdminRepo) ListCustomers(ctx context.Context, search, plan, status string, limit, offset int) ([]*domain.Customer, int, error) {
	baseWhere := "WHERE 1=1"
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
	if status == "suspended" {
		baseWhere += " AND suspended_at IS NOT NULL"
	} else if status == "active" {
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

	detail := &domain.AdminCustomerDetail{Customer: customer}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixels WHERE customer_id = $1`, id,
		).Scan(&detail.PixelCount)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events pe JOIN pixels p ON p.id = pe.pixel_id WHERE p.customer_id = $1`, id,
		).Scan(&detail.EventCount)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM sale_pages WHERE customer_id = $1`, id,
		).Scan(&detail.SalePageCount)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM replay_sessions WHERE customer_id = $1`, id,
		).Scan(&detail.ReplayCount)
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
			detail.Purchases = append(detail.Purchases, p)
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
			detail.Credits = append(detail.Credits, c)
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
			detail.Subscriptions = append(detail.Subscriptions, s)
		}
		return rows.Err()
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get customer detail: %w", err)
	}

	if detail.Purchases == nil {
		detail.Purchases = []*domain.Purchase{}
	}
	if detail.Credits == nil {
		detail.Credits = []*domain.ReplayCredit{}
	}
	if detail.Subscriptions == nil {
		detail.Subscriptions = []*domain.Subscription{}
	}

	return detail, nil
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

func (r *AdminRepo) GetPlatformStats(ctx context.Context) (*domain.PlatformStats, error) {
	stats := &domain.PlatformStats{
		CustomersByPlan: make(map[string]int),
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM customers`).Scan(&stats.TotalCustomers)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM customers WHERE suspended_at IS NULL`).Scan(&stats.ActiveCustomers)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM customers WHERE suspended_at IS NOT NULL`).Scan(&stats.SuspendedCustomers)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM pixels`).Scan(&stats.TotalPixels)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events WHERE created_at >= CURRENT_DATE`,
		).Scan(&stats.EventsToday)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events WHERE created_at >= date_trunc('month', CURRENT_DATE)`,
		).Scan(&stats.EventsThisMonth)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM replay_sessions`).Scan(&stats.TotalReplays)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM replay_sessions WHERE status = 'completed'`).Scan(&stats.SuccessfulReplays)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM replay_sessions WHERE status = 'failed'`).Scan(&stats.FailedReplays)
	})

	var totalRevenueSatang, monthRevenueSatang int64

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COALESCE(SUM(amount_satang), 0) FROM purchases WHERE status = 'completed'`,
		).Scan(&totalRevenueSatang)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COALESCE(SUM(amount_satang), 0) FROM purchases WHERE status = 'completed' AND completed_at >= date_trunc('month', CURRENT_DATE)`,
		).Scan(&monthRevenueSatang)
	})

	g.Go(func() error {
		rows, err := r.pool.Query(gCtx, `SELECT plan, COUNT(*) FROM customers GROUP BY plan`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var plan string
			var count int
			if err := rows.Scan(&plan, &count); err != nil {
				return err
			}
			stats.CustomersByPlan[plan] = count
		}
		return rows.Err()
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get platform stats: %w", err)
	}

	// Convert satang to THB
	stats.TotalRevenueTHB = float64(totalRevenueSatang) / 100
	stats.RevenueThisMonthTHB = float64(monthRevenueSatang) / 100

	return stats, nil
}

func (r *AdminRepo) GetRevenueChart(ctx context.Context, days int) ([]*domain.RevenueChartPoint, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT d::date::text AS date,
		        COALESCE(SUM(p.amount_satang), 0) AS amount_satang,
		        COUNT(p.id) AS purchase_count
		 FROM generate_series(CURRENT_DATE - ($1 - 1) * INTERVAL '1 day', CURRENT_DATE, '1 day') AS d
		 LEFT JOIN purchases p ON p.completed_at::date = d::date AND p.status = 'completed'
		 GROUP BY d::date
		 ORDER BY d::date`,
		days,
	)
	if err != nil {
		return nil, fmt.Errorf("get revenue chart: %w", err)
	}
	defer rows.Close()

	var points []*domain.RevenueChartPoint
	for rows.Next() {
		p := &domain.RevenueChartPoint{}
		if err := rows.Scan(&p.Date, &p.AmountSatang, &p.PurchaseCount); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (r *AdminRepo) GetGrowthChart(ctx context.Context, days int) ([]*domain.GrowthChartPoint, error) {
	rows, err := r.pool.Query(ctx,
		`WITH daily AS (
			SELECT d::date AS date,
			       COUNT(c.id) AS new_customers
			FROM generate_series(CURRENT_DATE - ($1 - 1) * INTERVAL '1 day', CURRENT_DATE, '1 day') AS d
			LEFT JOIN customers c ON c.created_at::date = d::date
			GROUP BY d::date
			ORDER BY d::date
		)
		SELECT date::text, new_customers,
		       SUM(new_customers) OVER (ORDER BY date) +
		         (SELECT COUNT(*) FROM customers WHERE created_at::date < CURRENT_DATE - ($1 - 1) * INTERVAL '1 day') AS total_customers
		FROM daily`,
		days,
	)
	if err != nil {
		return nil, fmt.Errorf("get growth chart: %w", err)
	}
	defer rows.Close()

	var points []*domain.GrowthChartPoint
	for rows.Next() {
		p := &domain.GrowthChartPoint{}
		if err := rows.Scan(&p.Date, &p.NewCustomers, &p.TotalCustomers); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

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

