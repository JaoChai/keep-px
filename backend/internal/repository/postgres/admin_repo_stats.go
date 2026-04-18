package postgres

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

func (r *AdminRepo) GetPlatformStats(ctx context.Context) (*domain.PlatformStats, error) {
	var (
		totalCustomers     int
		activeCustomers    int
		suspendedCustomers int
		totalPixels        int
		eventsToday        int64
		eventsThisMonth    int64
		totalReplays       int
		successfulReplays  int
		failedReplays      int
		totalRevenueSatang int64
		monthRevenueSatang int64
		customersByPlan    = make(map[string]int)
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM customers`).Scan(&totalCustomers)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM customers WHERE suspended_at IS NULL`).Scan(&activeCustomers)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM customers WHERE suspended_at IS NOT NULL`).Scan(&suspendedCustomers)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM pixels`).Scan(&totalPixels)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events WHERE created_at >= CURRENT_DATE`,
		).Scan(&eventsToday)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events WHERE created_at >= date_trunc('month', CURRENT_DATE)`,
		).Scan(&eventsThisMonth)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM replay_sessions`).Scan(&totalReplays)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM replay_sessions WHERE status = $1`, domain.ReplayStatusCompleted).Scan(&successfulReplays)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx, `SELECT COUNT(*) FROM replay_sessions WHERE status = $1`, domain.ReplayStatusFailed).Scan(&failedReplays)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COALESCE(SUM(amount_satang), 0) FROM purchases WHERE status = $1`, domain.PurchaseStatusCompleted,
		).Scan(&totalRevenueSatang)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COALESCE(SUM(amount_satang), 0) FROM purchases WHERE status = $1 AND completed_at >= date_trunc('month', CURRENT_DATE)`, domain.PurchaseStatusCompleted,
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
			customersByPlan[plan] = count
		}
		return rows.Err()
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get platform stats: %w", err)
	}

	return &domain.PlatformStats{
		TotalCustomers:     totalCustomers,
		ActiveCustomers:    activeCustomers,
		SuspendedCustomers: suspendedCustomers,
		TotalPixels:        totalPixels,
		EventsToday:        eventsToday,
		EventsThisMonth:    eventsThisMonth,
		TotalReplays:       totalReplays,
		SuccessfulReplays:  successfulReplays,
		FailedReplays:      failedReplays,
		TotalRevenueTHB:    float64(totalRevenueSatang) / 100,
		RevenueThisMonthTHB: float64(monthRevenueSatang) / 100,
		CustomersByPlan:    customersByPlan,
	}, nil
}

func (r *AdminRepo) GetRevenueChart(ctx context.Context, days int) ([]*domain.RevenueChartPoint, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT d::date::text AS date,
		        COALESCE(SUM(p.amount_satang), 0) AS amount_satang,
		        COUNT(p.id) AS purchase_count
		 FROM generate_series(CURRENT_DATE - ($1 - 1) * INTERVAL '1 day', CURRENT_DATE, '1 day') AS d
		 LEFT JOIN purchases p ON p.completed_at::date = d::date AND p.status = $2
		 GROUP BY d::date
		 ORDER BY d::date`,
		days, domain.PurchaseStatusCompleted,
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
