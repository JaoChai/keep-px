package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

type AnalyticsService struct {
	pool *pgxpool.Pool
}

func NewAnalyticsService(pool *pgxpool.Pool) *AnalyticsService {
	return &AnalyticsService{pool: pool}
}

type OverviewStats struct {
	TotalPixels     int `json:"total_pixels"`
	ActivePixels    int `json:"active_pixels"`
	TotalEvents     int `json:"total_events"`
	EventsToday     int `json:"events_today"`
	EventsYesterday int `json:"events_yesterday"`
	EventsThisWeek  int `json:"events_this_week"`
	TotalReplays    int `json:"total_replays"`
	ActiveReplays   int `json:"active_replays"`
	ForwardedEvents int `json:"forwarded_events"`
}

func (s *AnalyticsService) GetOverview(ctx context.Context, customerID string) (*OverviewStats, error) {
	stats := &OverviewStats{}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.pool.QueryRow(ctx,
			`SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active) FROM pixels WHERE customer_id = $1`, customerID,
		).Scan(&stats.TotalPixels, &stats.ActivePixels)
	})

	g.Go(func() error {
		return s.pool.QueryRow(ctx,
			`SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE pe.event_time >= CURRENT_DATE),
				COUNT(*) FILTER (WHERE pe.event_time >= CURRENT_DATE - INTERVAL '1 day'
				                   AND pe.event_time < CURRENT_DATE),
				COUNT(*) FILTER (WHERE pe.event_time >= CURRENT_DATE - INTERVAL '7 days'),
				COUNT(*) FILTER (WHERE pe.forwarded_to_capi)
			 FROM pixel_events pe
			 JOIN pixels p ON p.id = pe.pixel_id
			 WHERE p.customer_id = $1`, customerID,
		).Scan(&stats.TotalEvents, &stats.EventsToday, &stats.EventsYesterday, &stats.EventsThisWeek, &stats.ForwardedEvents)
	})

	g.Go(func() error {
		return s.pool.QueryRow(ctx,
			`SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE status IN ('running', 'pending'))
			 FROM replay_sessions WHERE customer_id = $1`, customerID,
		).Scan(&stats.TotalReplays, &stats.ActiveReplays)
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get overview: %w", err)
	}

	return stats, nil
}

type EventChartData struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

func (s *AnalyticsService) GetEventChart(ctx context.Context, customerID string, days int) ([]EventChartData, error) {
	if days < 1 || days > 90 {
		days = 30
	}

	rows, err := s.pool.Query(ctx,
		`SELECT DATE(pe.event_time) as date, COUNT(*) as count
		 FROM pixel_events pe
		 JOIN pixels p ON p.id = pe.pixel_id
		 WHERE p.customer_id = $1
		   AND pe.event_time >= CURRENT_DATE - make_interval(days => $2)
		 GROUP BY DATE(pe.event_time)
		 ORDER BY date`, customerID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query event chart: %w", err)
	}
	defer rows.Close()

	var data []EventChartData
	for rows.Next() {
		var d EventChartData
		var date interface{}
		if err := rows.Scan(&date, &d.Count); err != nil {
			return nil, err
		}
		d.Date = fmt.Sprintf("%v", date)
		data = append(data, d)
	}
	if data == nil {
		data = []EventChartData{}
	}
	return data, rows.Err()
}
