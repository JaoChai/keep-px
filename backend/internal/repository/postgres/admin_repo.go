package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

// F1: Sale Pages

func (r *AdminRepo) ListAllSalePages(ctx context.Context, search, customerID string, published *bool, limit, offset int) ([]*domain.AdminSalePage, int, error) {
	baseWhere := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (sp.name ILIKE $%d OR sp.slug ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if customerID != "" {
		baseWhere += fmt.Sprintf(" AND sp.customer_id = $%d", argIdx)
		args = append(args, customerID)
		argIdx++
	}
	if published != nil {
		baseWhere += fmt.Sprintf(" AND sp.is_published = $%d", argIdx)
		args = append(args, *published)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM sale_pages sp " + baseWhere
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count sale pages: %w", err)
	}

	selectQuery := fmt.Sprintf(
		`SELECT sp.id, sp.customer_id, sp.pixel_ids, sp.name, sp.slug, sp.template_name, sp.content, sp.is_published, sp.created_at, sp.updated_at,
		        c.email, c.name,
		        (SELECT COUNT(*) FROM pixel_events pe JOIN pixels p ON p.id = pe.pixel_id WHERE p.id = ANY(sp.pixel_ids)) AS event_count
		 FROM sale_pages sp JOIN customers c ON c.id = sp.customer_id
		 %s ORDER BY sp.created_at DESC LIMIT $%d OFFSET $%d`,
		baseWhere, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list sale pages: %w", err)
	}
	defer rows.Close()

	var pages []*domain.AdminSalePage
	for rows.Next() {
		sp := &domain.AdminSalePage{}
		if err := rows.Scan(
			&sp.ID, &sp.CustomerID, &sp.PixelIDs, &sp.Name, &sp.Slug, &sp.TemplateName, &sp.Content, &sp.IsPublished, &sp.CreatedAt, &sp.UpdatedAt,
			&sp.CustomerEmail, &sp.CustomerName,
			&sp.EventCount,
		); err != nil {
			return nil, 0, err
		}
		pages = append(pages, sp)
	}
	if pages == nil {
		pages = []*domain.AdminSalePage{}
	}
	return pages, total, rows.Err()
}

func (r *AdminRepo) GetSalePageAdminDetail(ctx context.Context, id string) (*domain.AdminSalePageDetail, error) {
	detail := &domain.AdminSalePageDetail{SalePage: &domain.SalePage{}}

	err := r.pool.QueryRow(ctx,
		`SELECT sp.id, sp.customer_id, sp.pixel_ids, sp.name, sp.slug, sp.template_name, sp.content, sp.is_published, sp.created_at, sp.updated_at,
		        c.email, c.name
		 FROM sale_pages sp JOIN customers c ON c.id = sp.customer_id
		 WHERE sp.id = $1`, id,
	).Scan(
		&detail.SalePage.ID, &detail.SalePage.CustomerID, &detail.SalePage.PixelIDs, &detail.SalePage.Name, &detail.SalePage.Slug,
		&detail.SalePage.TemplateName, &detail.SalePage.Content, &detail.SalePage.IsPublished, &detail.SalePage.CreatedAt, &detail.SalePage.UpdatedAt,
		&detail.CustomerEmail, &detail.CustomerName,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get sale page detail: %w", err)
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if len(detail.SalePage.PixelIDs) == 0 {
			detail.LinkedPixels = []*domain.Pixel{}
			return nil
		}
		rows, err := r.pool.Query(gCtx,
			`SELECT id, customer_id, fb_pixel_id, name, is_active, status, backup_pixel_id, test_event_code, created_at, updated_at
			 FROM pixels WHERE id = ANY($1)`, detail.SalePage.PixelIDs,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			p := &domain.Pixel{}
			if err := rows.Scan(&p.ID, &p.CustomerID, &p.FBPixelID, &p.Name, &p.IsActive, &p.Status, &p.BackupPixelID, &p.TestEventCode, &p.CreatedAt, &p.UpdatedAt); err != nil {
				return err
			}
			detail.LinkedPixels = append(detail.LinkedPixels, p)
		}
		return rows.Err()
	})

	g.Go(func() error {
		if len(detail.SalePage.PixelIDs) == 0 {
			return nil
		}
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events WHERE pixel_id = ANY($1)`, detail.SalePage.PixelIDs,
		).Scan(&detail.EventCount)
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get sale page detail sub-queries: %w", err)
	}

	if detail.LinkedPixels == nil {
		detail.LinkedPixels = []*domain.Pixel{}
	}

	return detail, nil
}

func (r *AdminRepo) SetSalePagePublished(ctx context.Context, id string, published bool) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE sale_pages SET is_published = $2, updated_at = NOW() WHERE id = $1`, id, published,
	)
	if err != nil {
		return fmt.Errorf("set sale page published: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *AdminRepo) DeleteSalePageByAdmin(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM sale_pages WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete sale page: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// F2: Pixels

func (r *AdminRepo) ListAllPixels(ctx context.Context, search, customerID string, active *bool, limit, offset int) ([]*domain.AdminPixel, int, error) {
	baseWhere := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.fb_pixel_id ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if customerID != "" {
		baseWhere += fmt.Sprintf(" AND p.customer_id = $%d", argIdx)
		args = append(args, customerID)
		argIdx++
	}
	if active != nil {
		baseWhere += fmt.Sprintf(" AND p.is_active = $%d", argIdx)
		args = append(args, *active)
		argIdx++
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM pixels p "+baseWhere, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count pixels: %w", err)
	}

	selectQuery := fmt.Sprintf(
		`SELECT p.id, p.customer_id, p.fb_pixel_id, p.name, p.is_active, p.status, p.backup_pixel_id, p.test_event_code, p.created_at, p.updated_at,
		        c.email, c.name,
		        (SELECT COUNT(*) FROM pixel_events pe WHERE pe.pixel_id = p.id) AS event_count,
		        (SELECT COUNT(*) FROM sale_pages sp WHERE p.id = ANY(sp.pixel_ids)) AS sale_page_count
		 FROM pixels p JOIN customers c ON c.id = p.customer_id
		 %s ORDER BY p.created_at DESC LIMIT $%d OFFSET $%d`,
		baseWhere, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list pixels: %w", err)
	}
	defer rows.Close()

	var pixels []*domain.AdminPixel
	for rows.Next() {
		ap := &domain.AdminPixel{}
		if err := rows.Scan(
			&ap.ID, &ap.CustomerID, &ap.FBPixelID, &ap.Name, &ap.IsActive, &ap.Status, &ap.BackupPixelID, &ap.TestEventCode, &ap.CreatedAt, &ap.UpdatedAt,
			&ap.CustomerEmail, &ap.CustomerName,
			&ap.EventCount, &ap.SalePageCount,
		); err != nil {
			return nil, 0, err
		}
		pixels = append(pixels, ap)
	}
	if pixels == nil {
		pixels = []*domain.AdminPixel{}
	}
	return pixels, total, rows.Err()
}

func (r *AdminRepo) GetPixelAdminDetail(ctx context.Context, id string) (*domain.AdminPixelDetail, error) {
	detail := &domain.AdminPixelDetail{Pixel: &domain.Pixel{}}

	err := r.pool.QueryRow(ctx,
		`SELECT p.id, p.customer_id, p.fb_pixel_id, p.name, p.is_active, p.status, p.backup_pixel_id, p.test_event_code, p.created_at, p.updated_at,
		        c.email, c.name
		 FROM pixels p JOIN customers c ON c.id = p.customer_id
		 WHERE p.id = $1`, id,
	).Scan(
		&detail.Pixel.ID, &detail.Pixel.CustomerID, &detail.Pixel.FBPixelID, &detail.Pixel.Name,
		&detail.Pixel.IsActive, &detail.Pixel.Status, &detail.Pixel.BackupPixelID, &detail.Pixel.TestEventCode,
		&detail.Pixel.CreatedAt, &detail.Pixel.UpdatedAt,
		&detail.CustomerEmail, &detail.CustomerName,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get pixel detail: %w", err)
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events WHERE pixel_id = $1`, id,
		).Scan(&detail.EventCount)
	})

	g.Go(func() error {
		rows, err := r.pool.Query(gCtx,
			`SELECT id, customer_id, pixel_ids, name, slug, template_name, content, is_published, created_at, updated_at
			 FROM sale_pages WHERE $1 = ANY(pixel_ids)`, id,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			sp := &domain.SalePage{}
			if err := rows.Scan(&sp.ID, &sp.CustomerID, &sp.PixelIDs, &sp.Name, &sp.Slug, &sp.TemplateName, &sp.Content, &sp.IsPublished, &sp.CreatedAt, &sp.UpdatedAt); err != nil {
				return err
			}
			detail.LinkedSalePages = append(detail.LinkedSalePages, sp)
		}
		return rows.Err()
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get pixel detail sub-queries: %w", err)
	}

	if detail.LinkedSalePages == nil {
		detail.LinkedSalePages = []*domain.SalePage{}
	}

	return detail, nil
}

func (r *AdminRepo) SetPixelActive(ctx context.Context, id string, active bool) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE pixels SET is_active = $2, updated_at = NOW() WHERE id = $1`, id, active,
	)
	if err != nil {
		return fmt.Errorf("set pixel active: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// F3: Replays

func (r *AdminRepo) ListAllReplaySessions(ctx context.Context, status, customerID string, limit, offset int) ([]*domain.AdminReplaySession, int, error) {
	baseWhere := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if status != "" {
		baseWhere += fmt.Sprintf(" AND rs.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if customerID != "" {
		baseWhere += fmt.Sprintf(" AND rs.customer_id = $%d", argIdx)
		args = append(args, customerID)
		argIdx++
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM replay_sessions rs "+baseWhere, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count replay sessions: %w", err)
	}

	selectQuery := fmt.Sprintf(
		`SELECT rs.id, rs.customer_id, rs.source_pixel_id, rs.target_pixel_id, rs.status,
		        rs.total_events, rs.replayed_events, rs.failed_events,
		        rs.event_types, rs.date_from, rs.date_to, rs.time_mode, rs.batch_delay_ms,
		        rs.error_message, rs.started_at, rs.completed_at, rs.cancelled_at, rs.failed_batch_ranges, rs.created_at,
		        c.email, c.name,
		        COALESCE(sp.name, '') AS source_pixel_name,
		        COALESCE(tp.name, '') AS target_pixel_name
		 FROM replay_sessions rs
		 JOIN customers c ON c.id = rs.customer_id
		 LEFT JOIN pixels sp ON sp.id = rs.source_pixel_id
		 LEFT JOIN pixels tp ON tp.id = rs.target_pixel_id
		 %s ORDER BY rs.created_at DESC LIMIT $%d OFFSET $%d`,
		baseWhere, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list replay sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.AdminReplaySession
	for rows.Next() {
		s := &domain.AdminReplaySession{}
		if err := rows.Scan(
			&s.ID, &s.CustomerID, &s.SourcePixelID, &s.TargetPixelID, &s.Status,
			&s.TotalEvents, &s.ReplayedEvents, &s.FailedEvents,
			&s.EventTypes, &s.DateFrom, &s.DateTo, &s.TimeMode, &s.BatchDelayMs,
			&s.ErrorMessage, &s.StartedAt, &s.CompletedAt, &s.CancelledAt, &s.FailedBatchRanges, &s.CreatedAt,
			&s.CustomerEmail, &s.CustomerName,
			&s.SourcePixelName, &s.TargetPixelName,
		); err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, s)
	}
	if sessions == nil {
		sessions = []*domain.AdminReplaySession{}
	}
	return sessions, total, rows.Err()
}

func (r *AdminRepo) GetReplaySessionAdminDetail(ctx context.Context, id string) (*domain.AdminReplaySessionDetail, error) {
	detail := &domain.AdminReplaySessionDetail{Session: &domain.ReplaySession{}}

	var sourcePixelID, targetPixelID string
	err := r.pool.QueryRow(ctx,
		`SELECT rs.id, rs.customer_id, rs.source_pixel_id, rs.target_pixel_id, rs.status,
		        rs.total_events, rs.replayed_events, rs.failed_events,
		        rs.event_types, rs.date_from, rs.date_to, rs.time_mode, rs.batch_delay_ms,
		        rs.error_message, rs.started_at, rs.completed_at, rs.cancelled_at, rs.failed_batch_ranges, rs.created_at,
		        c.email, c.name
		 FROM replay_sessions rs JOIN customers c ON c.id = rs.customer_id
		 WHERE rs.id = $1`, id,
	).Scan(
		&detail.Session.ID, &detail.Session.CustomerID, &detail.Session.SourcePixelID, &detail.Session.TargetPixelID, &detail.Session.Status,
		&detail.Session.TotalEvents, &detail.Session.ReplayedEvents, &detail.Session.FailedEvents,
		&detail.Session.EventTypes, &detail.Session.DateFrom, &detail.Session.DateTo, &detail.Session.TimeMode, &detail.Session.BatchDelayMs,
		&detail.Session.ErrorMessage, &detail.Session.StartedAt, &detail.Session.CompletedAt, &detail.Session.CancelledAt, &detail.Session.FailedBatchRanges, &detail.Session.CreatedAt,
		&detail.CustomerEmail, &detail.CustomerName,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get replay detail: %w", err)
	}

	sourcePixelID = detail.Session.SourcePixelID
	targetPixelID = detail.Session.TargetPixelID

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		p := &domain.Pixel{}
		err := r.pool.QueryRow(gCtx,
			`SELECT id, customer_id, fb_pixel_id, name, is_active, status, backup_pixel_id, test_event_code, created_at, updated_at
			 FROM pixels WHERE id = $1`, sourcePixelID,
		).Scan(&p.ID, &p.CustomerID, &p.FBPixelID, &p.Name, &p.IsActive, &p.Status, &p.BackupPixelID, &p.TestEventCode, &p.CreatedAt, &p.UpdatedAt)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		detail.SourcePixel = p
		return nil
	})

	g.Go(func() error {
		p := &domain.Pixel{}
		err := r.pool.QueryRow(gCtx,
			`SELECT id, customer_id, fb_pixel_id, name, is_active, status, backup_pixel_id, test_event_code, created_at, updated_at
			 FROM pixels WHERE id = $1`, targetPixelID,
		).Scan(&p.ID, &p.CustomerID, &p.FBPixelID, &p.Name, &p.IsActive, &p.Status, &p.BackupPixelID, &p.TestEventCode, &p.CreatedAt, &p.UpdatedAt)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		detail.TargetPixel = p
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get replay detail sub-queries: %w", err)
	}

	return detail, nil
}

// F4: Events

func (r *AdminRepo) ListAllEvents(ctx context.Context, customerID, pixelID, eventName string, limit, offset int) ([]*domain.AdminEvent, int, error) {
	baseWhere := "WHERE pe.created_at > NOW() - INTERVAL '24 hours'"
	args := []interface{}{}
	argIdx := 1

	if customerID != "" {
		baseWhere += fmt.Sprintf(" AND p.customer_id = $%d", argIdx)
		args = append(args, customerID)
		argIdx++
	}
	if pixelID != "" {
		baseWhere += fmt.Sprintf(" AND pe.pixel_id = $%d", argIdx)
		args = append(args, pixelID)
		argIdx++
	}
	if eventName != "" {
		baseWhere += fmt.Sprintf(" AND pe.event_name = $%d", argIdx)
		args = append(args, eventName)
		argIdx++
	}

	var total int
	countQuery := fmt.Sprintf(
		`SELECT COUNT(*) FROM pixel_events pe JOIN pixels p ON p.id = pe.pixel_id %s`, baseWhere,
	)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count events: %w", err)
	}

	selectQuery := fmt.Sprintf(
		`SELECT pe.id, pe.pixel_id, pe.event_name, pe.event_data, pe.user_data, pe.source_url, pe.event_id,
		        pe.client_ip, pe.client_user_agent, pe.event_time, pe.forwarded_to_capi, pe.capi_response_code, pe.capi_events_received, pe.created_at,
		        p.name AS pixel_name,
		        c.email, c.name
		 FROM pixel_events pe
		 JOIN pixels p ON p.id = pe.pixel_id
		 JOIN customers c ON c.id = p.customer_id
		 %s ORDER BY pe.created_at DESC LIMIT $%d OFFSET $%d`,
		baseWhere, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []*domain.AdminEvent
	for rows.Next() {
		e := &domain.AdminEvent{}
		if err := rows.Scan(
			&e.ID, &e.PixelID, &e.EventName, &e.EventData, &e.UserData, &e.SourceURL, &e.EventID,
			&e.ClientIP, &e.ClientUserAgent, &e.EventTime, &e.ForwardedToCAPI, &e.CAPIResponseCode, &e.CAPIEventsReceived, &e.CreatedAt,
			&e.PixelName,
			&e.CustomerEmail, &e.CustomerName,
		); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	if events == nil {
		events = []*domain.AdminEvent{}
	}
	return events, total, rows.Err()
}

func (r *AdminRepo) GetEventStats(ctx context.Context, hours int) (*domain.AdminEventStats, error) {
	stats := &domain.AdminEventStats{}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events WHERE created_at >= CURRENT_DATE`,
		).Scan(&stats.TotalToday)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events WHERE created_at >= date_trunc('hour', NOW())`,
		).Scan(&stats.TotalThisHour)
	})

	g.Go(func() error {
		var total, success int64
		err := r.pool.QueryRow(gCtx,
			`SELECT COUNT(*), COUNT(*) FILTER (WHERE forwarded_to_capi = true AND capi_response_code = 200)
			 FROM pixel_events WHERE created_at >= CURRENT_DATE`,
		).Scan(&total, &success)
		if err != nil {
			return err
		}
		if total > 0 {
			stats.CAPISuccessRate = float64(success) / float64(total) * 100
		}
		stats.CAPIFailureCount = total - success
		return nil
	})

	g.Go(func() error {
		rows, err := r.pool.Query(gCtx,
			`SELECT event_name, COUNT(*) AS cnt
			 FROM pixel_events WHERE created_at >= CURRENT_DATE
			 GROUP BY event_name ORDER BY cnt DESC LIMIT 10`,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			tc := domain.EventTypeCount{}
			if err := rows.Scan(&tc.EventName, &tc.Count); err != nil {
				return err
			}
			stats.TopEventTypes = append(stats.TopEventTypes, tc)
		}
		return rows.Err()
	})

	g.Go(func() error {
		rows, err := r.pool.Query(gCtx,
			`SELECT date_trunc('hour', created_at)::text AS ts,
			        COUNT(*) AS event_count,
			        COUNT(*) FILTER (WHERE forwarded_to_capi = true AND capi_response_code = 200) AS capi_success,
			        COUNT(*) FILTER (WHERE forwarded_to_capi = true AND (capi_response_code IS NULL OR capi_response_code != 200)) AS capi_failure
			 FROM pixel_events
			 WHERE created_at >= NOW() - make_interval(hours => $1)
			 GROUP BY ts ORDER BY ts`, hours,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			tp := domain.EventTimeseriesPoint{}
			if err := rows.Scan(&tp.Timestamp, &tp.EventCount, &tp.CAPISuccess, &tp.CAPIFailure); err != nil {
				return err
			}
			stats.Timeseries = append(stats.Timeseries, tp)
		}
		return rows.Err()
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get event stats: %w", err)
	}

	if stats.TopEventTypes == nil {
		stats.TopEventTypes = []domain.EventTypeCount{}
	}
	if stats.Timeseries == nil {
		stats.Timeseries = []domain.EventTimeseriesPoint{}
	}

	return stats, nil
}

// F5: Audit Log

func (r *AdminRepo) CreateAuditLog(ctx context.Context, entry *domain.AuditLogEntry) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO admin_audit_logs (admin_id, action, target_type, target_id, target_customer_id, details)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		entry.AdminID, entry.Action, entry.TargetType, entry.TargetID, entry.TargetCustomerID, entry.Details,
	).Scan(&entry.ID, &entry.CreatedAt)
}

func (r *AdminRepo) ListAuditLogs(ctx context.Context, adminID, action, targetCustomerID string, from, to *time.Time, limit, offset int) ([]*domain.AuditLogEntry, int, error) {
	baseWhere := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if adminID != "" {
		baseWhere += fmt.Sprintf(" AND al.admin_id = $%d", argIdx)
		args = append(args, adminID)
		argIdx++
	}
	if action != "" {
		baseWhere += fmt.Sprintf(" AND al.action = $%d", argIdx)
		args = append(args, action)
		argIdx++
	}
	if targetCustomerID != "" {
		baseWhere += fmt.Sprintf(" AND al.target_customer_id = $%d", argIdx)
		args = append(args, targetCustomerID)
		argIdx++
	}
	if from != nil {
		baseWhere += fmt.Sprintf(" AND al.created_at >= $%d", argIdx)
		args = append(args, *from)
		argIdx++
	}
	if to != nil {
		baseWhere += fmt.Sprintf(" AND al.created_at <= $%d", argIdx)
		args = append(args, *to)
		argIdx++
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM admin_audit_logs al "+baseWhere, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	selectQuery := fmt.Sprintf(
		`SELECT al.id, al.admin_id, admin_c.email, al.action, al.target_type, al.target_id, al.target_customer_id,
		        COALESCE(target_c.email, ''), al.details, al.created_at
		 FROM admin_audit_logs al
		 JOIN customers admin_c ON admin_c.id = al.admin_id
		 LEFT JOIN customers target_c ON target_c.id = al.target_customer_id
		 %s ORDER BY al.created_at DESC LIMIT $%d OFFSET $%d`,
		baseWhere, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var entries []*domain.AuditLogEntry
	for rows.Next() {
		e := &domain.AuditLogEntry{}
		var details json.RawMessage
		if err := rows.Scan(
			&e.ID, &e.AdminID, &e.AdminEmail, &e.Action, &e.TargetType, &e.TargetID, &e.TargetCustomerID,
			&e.CustomerEmail, &details, &e.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		e.Details = details
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []*domain.AuditLogEntry{}
	}
	return entries, total, rows.Err()
}

