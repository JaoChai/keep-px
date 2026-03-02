package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type EventRepo struct {
	pool *pgxpool.Pool
}

func NewEventRepo(pool *pgxpool.Pool) *EventRepo {
	return &EventRepo{pool: pool}
}

func (r *EventRepo) Create(ctx context.Context, e *domain.PixelEvent) (bool, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO pixel_events (pixel_id, event_name, event_data, user_data, source_url, event_time, event_id, client_ip, client_user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (pixel_id, event_id) WHERE event_id IS NOT NULL DO NOTHING
		 RETURNING id, forwarded_to_capi, created_at`,
		e.PixelID, e.EventName, e.EventData, e.UserData, e.SourceURL, e.EventTime, e.EventID, e.ClientIP, e.ClientUserAgent,
	).Scan(&e.ID, &e.ForwardedToCAPI, &e.CreatedAt)
	if err == pgx.ErrNoRows {
		return false, nil // duplicate, skipped
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *EventRepo) GetByID(ctx context.Context, id string) (*domain.PixelEvent, error) {
	e := &domain.PixelEvent{}
	var eventID, clientIP, clientUA *string
	err := r.pool.QueryRow(ctx,
		`SELECT id, pixel_id, event_name, event_data, user_data, source_url, event_time, forwarded_to_capi, capi_response_code, capi_events_received, created_at, event_id, client_ip, client_user_agent
		 FROM pixel_events WHERE id = $1`, id,
	).Scan(&e.ID, &e.PixelID, &e.EventName, &e.EventData, &e.UserData, &e.SourceURL, &e.EventTime, &e.ForwardedToCAPI, &e.CAPIResponseCode, &e.CAPIEventsReceived, &e.CreatedAt, &eventID, &clientIP, &clientUA)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if eventID != nil {
		e.EventID = *eventID
	}
	if clientIP != nil {
		e.ClientIP = *clientIP
	}
	if clientUA != nil {
		e.ClientUserAgent = *clientUA
	}
	return e, nil
}

func (r *EventRepo) ListByPixelID(ctx context.Context, pixelID string, limit, offset int) ([]*domain.PixelEvent, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pixel_events WHERE pixel_id = $1`, pixelID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, pixel_id, event_name, event_data, user_data, source_url, event_time, forwarded_to_capi, capi_response_code, capi_events_received, created_at, event_id, client_ip, client_user_agent
		 FROM pixel_events WHERE pixel_id = $1
		 ORDER BY event_time DESC LIMIT $2 OFFSET $3`, pixelID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*domain.PixelEvent
	for rows.Next() {
		e := &domain.PixelEvent{}
		var eventID, clientIP, clientUA *string
		if err := rows.Scan(&e.ID, &e.PixelID, &e.EventName, &e.EventData, &e.UserData, &e.SourceURL, &e.EventTime, &e.ForwardedToCAPI, &e.CAPIResponseCode, &e.CAPIEventsReceived, &e.CreatedAt, &eventID, &clientIP, &clientUA); err != nil {
			return nil, 0, err
		}
		if eventID != nil {
			e.EventID = *eventID
		}
		if clientIP != nil {
			e.ClientIP = *clientIP
		}
		if clientUA != nil {
			e.ClientUserAgent = *clientUA
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

func (r *EventRepo) ListByCustomerID(ctx context.Context, customerID string, pixelID string, limit, offset int) ([]*domain.PixelEvent, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pixel_events pe JOIN pixels p ON p.id = pe.pixel_id
		 WHERE p.customer_id = $1 AND ($2::text = '' OR pe.pixel_id = $2::uuid)`, customerID, pixelID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT pe.id, pe.pixel_id, pe.event_name, pe.event_data, pe.user_data, pe.source_url, pe.event_time, pe.forwarded_to_capi, pe.capi_response_code, pe.capi_events_received, pe.created_at, pe.event_id, pe.client_ip, pe.client_user_agent
		 FROM pixel_events pe JOIN pixels p ON p.id = pe.pixel_id
		 WHERE p.customer_id = $1
		   AND ($2::text = '' OR pe.pixel_id = $2::uuid)
		 ORDER BY pe.event_time DESC LIMIT $3 OFFSET $4`, customerID, pixelID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*domain.PixelEvent
	for rows.Next() {
		e := &domain.PixelEvent{}
		var eventID, clientIP, clientUA *string
		if err := rows.Scan(&e.ID, &e.PixelID, &e.EventName, &e.EventData, &e.UserData, &e.SourceURL, &e.EventTime, &e.ForwardedToCAPI, &e.CAPIResponseCode, &e.CAPIEventsReceived, &e.CreatedAt, &eventID, &clientIP, &clientUA); err != nil {
			return nil, 0, err
		}
		if eventID != nil {
			e.EventID = *eventID
		}
		if clientIP != nil {
			e.ClientIP = *clientIP
		}
		if clientUA != nil {
			e.ClientUserAgent = *clientUA
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

func (r *EventRepo) MarkForwarded(ctx context.Context, id string, responseCode int, eventsReceived int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE pixel_events SET forwarded_to_capi = true, capi_response_code = $2, capi_events_received = $3 WHERE id = $1`,
		id, responseCode, eventsReceived,
	)
	return err
}

func (r *EventRepo) GetEventsForReplay(ctx context.Context, pixelID string, eventTypes []string, from, to *time.Time, createdBefore *time.Time) ([]*domain.PixelEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, pixel_id, event_name, event_data, user_data, source_url, event_time, forwarded_to_capi, capi_response_code, capi_events_received, created_at, event_id, client_ip, client_user_agent
		 FROM pixel_events
		 WHERE pixel_id = $1
		   AND ($2::text[] IS NULL OR event_name = ANY($2::text[]))
		   AND ($3::timestamptz IS NULL OR event_time >= $3)
		   AND ($4::timestamptz IS NULL OR event_time <= $4)
		   AND ($5::timestamptz IS NULL OR created_at < $5)
		 ORDER BY event_time ASC
		 LIMIT 100000`, // must match service.MaxReplayEvents
		pixelID, eventTypes, from, to, createdBefore,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.PixelEvent
	for rows.Next() {
		e := &domain.PixelEvent{}
		var eventID, clientIP, clientUA *string
		if err := rows.Scan(&e.ID, &e.PixelID, &e.EventName, &e.EventData, &e.UserData, &e.SourceURL, &e.EventTime, &e.ForwardedToCAPI, &e.CAPIResponseCode, &e.CAPIEventsReceived, &e.CreatedAt, &eventID, &clientIP, &clientUA); err != nil {
			return nil, err
		}
		if eventID != nil {
			e.EventID = *eventID
		}
		if clientIP != nil {
			e.ClientIP = *clientIP
		}
		if clientUA != nil {
			e.ClientUserAgent = *clientUA
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *EventRepo) CountEventsForReplay(ctx context.Context, pixelID string, eventTypes []string, from, to *time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM (
		   SELECT 1 FROM pixel_events
		   WHERE pixel_id = $1
		     AND ($2::text[] IS NULL OR event_name = ANY($2::text[]))
		     AND ($3::timestamptz IS NULL OR event_time >= $3)
		     AND ($4::timestamptz IS NULL OR event_time <= $4)
		   LIMIT 100001
		 ) sub`, // must match service.MaxReplayEvents + 1
		pixelID, eventTypes, from, to,
	).Scan(&count)
	return count, err
}

func (r *EventRepo) GetEventsForReplayPreview(ctx context.Context, pixelID string, eventTypes []string, from, to *time.Time, limit int) ([]*domain.PixelEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, pixel_id, event_name, event_data, user_data, source_url, event_time, forwarded_to_capi, capi_response_code, capi_events_received, created_at, event_id, client_ip, client_user_agent
		 FROM pixel_events
		 WHERE pixel_id = $1
		   AND ($2::text[] IS NULL OR event_name = ANY($2::text[]))
		   AND ($3::timestamptz IS NULL OR event_time >= $3)
		   AND ($4::timestamptz IS NULL OR event_time <= $4)
		 ORDER BY event_time ASC
		 LIMIT $5`,
		pixelID, eventTypes, from, to, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.PixelEvent
	for rows.Next() {
		e := &domain.PixelEvent{}
		var eventID, clientIP, clientUA *string
		if err := rows.Scan(&e.ID, &e.PixelID, &e.EventName, &e.EventData, &e.UserData, &e.SourceURL, &e.EventTime, &e.ForwardedToCAPI, &e.CAPIResponseCode, &e.CAPIEventsReceived, &e.CreatedAt, &eventID, &clientIP, &clientUA); err != nil {
			return nil, err
		}
		if eventID != nil {
			e.EventID = *eventID
		}
		if clientIP != nil {
			e.ClientIP = *clientIP
		}
		if clientUA != nil {
			e.ClientUserAgent = *clientUA
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *EventRepo) GetDistinctEventTypes(ctx context.Context, pixelID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT event_name FROM pixel_events WHERE pixel_id = $1 ORDER BY event_name`, pixelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var types []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, rows.Err()
}

func (r *EventRepo) ListLatestByCustomerID(ctx context.Context, customerID string, pixelID string, limit int) ([]*domain.RealtimeEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT pe.id, pe.pixel_id, p.name AS pixel_name, pe.event_name,
		        pe.source_url, pe.forwarded_to_capi, pe.event_time, pe.created_at
		 FROM pixel_events pe
		 JOIN pixels p ON p.id = pe.pixel_id
		 WHERE p.customer_id = $1
		   AND ($2::text = '' OR pe.pixel_id = $2::uuid)
		 ORDER BY pe.created_at DESC
		 LIMIT $3`,
		customerID, pixelID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.RealtimeEvent
	for rows.Next() {
		e := &domain.RealtimeEvent{}
		var sourceURL *string
		if err := rows.Scan(&e.ID, &e.PixelID, &e.PixelName, &e.EventName, &sourceURL, &e.ForwardedToCAPI, &e.EventTime, &e.CreatedAt); err != nil {
			return nil, err
		}
		if sourceURL != nil {
			e.SourceURL = *sourceURL
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Reverse to ASC order so frontend can use last event as cursor
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	return events, nil
}

func (r *EventRepo) ListRecentByCustomerID(ctx context.Context, customerID string, since time.Time, pixelID string, limit int) ([]*domain.RealtimeEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT pe.id, pe.pixel_id, p.name AS pixel_name, pe.event_name,
		        pe.source_url, pe.forwarded_to_capi, pe.event_time, pe.created_at
		 FROM pixel_events pe
		 JOIN pixels p ON p.id = pe.pixel_id
		 WHERE p.customer_id = $1
		   AND pe.created_at > $2
		   AND ($3::text = '' OR pe.pixel_id = $3::uuid)
		 ORDER BY pe.created_at ASC
		 LIMIT $4`,
		customerID, since, pixelID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.RealtimeEvent
	for rows.Next() {
		e := &domain.RealtimeEvent{}
		var sourceURL *string
		if err := rows.Scan(&e.ID, &e.PixelID, &e.PixelName, &e.EventName, &sourceURL, &e.ForwardedToCAPI, &e.EventTime, &e.CreatedAt); err != nil {
			return nil, err
		}
		if sourceURL != nil {
			e.SourceURL = *sourceURL
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
