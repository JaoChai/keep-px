package postgres

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

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
	var (
		totalToday       int64
		totalThisHour    int64
		capiSuccessRate  float64
		capiFailureCount int64
		topEventTypes    []domain.EventTypeCount
		timeseries       []domain.EventTimeseriesPoint
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events WHERE created_at >= CURRENT_DATE`,
		).Scan(&totalToday)
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events WHERE created_at >= date_trunc('hour', NOW())`,
		).Scan(&totalThisHour)
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
			capiSuccessRate = float64(success) / float64(total) * 100
		}
		capiFailureCount = total - success
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
			topEventTypes = append(topEventTypes, tc)
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
			timeseries = append(timeseries, tp)
		}
		return rows.Err()
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get event stats: %w", err)
	}

	if topEventTypes == nil {
		topEventTypes = []domain.EventTypeCount{}
	}
	if timeseries == nil {
		timeseries = []domain.EventTimeseriesPoint{}
	}

	return &domain.AdminEventStats{
		TotalToday:       totalToday,
		TotalThisHour:    totalThisHour,
		CAPISuccessRate:  capiSuccessRate,
		CAPIFailureCount: capiFailureCount,
		TopEventTypes:    topEventTypes,
		Timeseries:       timeseries,
	}, nil
}
