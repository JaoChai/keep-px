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

func (r *EventRepo) Create(ctx context.Context, e *domain.PixelEvent) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO pixel_events (pixel_id, event_name, event_data, user_data, source_url, event_time)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, forwarded_to_capi, created_at`,
		e.PixelID, e.EventName, e.EventData, e.UserData, e.SourceURL, e.EventTime,
	).Scan(&e.ID, &e.ForwardedToCAPI, &e.CreatedAt)
}

func (r *EventRepo) GetByID(ctx context.Context, id string) (*domain.PixelEvent, error) {
	e := &domain.PixelEvent{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, pixel_id, event_name, event_data, user_data, source_url, event_time, forwarded_to_capi, capi_response_code, created_at
		 FROM pixel_events WHERE id = $1`, id,
	).Scan(&e.ID, &e.PixelID, &e.EventName, &e.EventData, &e.UserData, &e.SourceURL, &e.EventTime, &e.ForwardedToCAPI, &e.CAPIResponseCode, &e.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return e, err
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
		`SELECT id, pixel_id, event_name, event_data, user_data, source_url, event_time, forwarded_to_capi, capi_response_code, created_at
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
		if err := rows.Scan(&e.ID, &e.PixelID, &e.EventName, &e.EventData, &e.UserData, &e.SourceURL, &e.EventTime, &e.ForwardedToCAPI, &e.CAPIResponseCode, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

func (r *EventRepo) ListByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]*domain.PixelEvent, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pixel_events pe JOIN pixels p ON p.id = pe.pixel_id WHERE p.customer_id = $1`, customerID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT pe.id, pe.pixel_id, pe.event_name, pe.event_data, pe.user_data, pe.source_url, pe.event_time, pe.forwarded_to_capi, pe.capi_response_code, pe.created_at
		 FROM pixel_events pe JOIN pixels p ON p.id = pe.pixel_id
		 WHERE p.customer_id = $1
		 ORDER BY pe.event_time DESC LIMIT $2 OFFSET $3`, customerID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*domain.PixelEvent
	for rows.Next() {
		e := &domain.PixelEvent{}
		if err := rows.Scan(&e.ID, &e.PixelID, &e.EventName, &e.EventData, &e.UserData, &e.SourceURL, &e.EventTime, &e.ForwardedToCAPI, &e.CAPIResponseCode, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

func (r *EventRepo) MarkForwarded(ctx context.Context, id string, responseCode int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE pixel_events SET forwarded_to_capi = true, capi_response_code = $2 WHERE id = $1`,
		id, responseCode,
	)
	return err
}

func (r *EventRepo) GetEventsForReplay(ctx context.Context, pixelID string, eventTypes []string, from, to *time.Time) ([]*domain.PixelEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, pixel_id, event_name, event_data, user_data, source_url, event_time, forwarded_to_capi, capi_response_code, created_at
		 FROM pixel_events
		 WHERE pixel_id = $1
		   AND ($2::text[] IS NULL OR event_name = ANY($2::text[]))
		   AND ($3::timestamptz IS NULL OR event_time >= $3)
		   AND ($4::timestamptz IS NULL OR event_time <= $4)
		 ORDER BY event_time ASC`,
		pixelID, eventTypes, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.PixelEvent
	for rows.Next() {
		e := &domain.PixelEvent{}
		if err := rows.Scan(&e.ID, &e.PixelID, &e.EventName, &e.EventData, &e.UserData, &e.SourceURL, &e.EventTime, &e.ForwardedToCAPI, &e.CAPIResponseCode, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
