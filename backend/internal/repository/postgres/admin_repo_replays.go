package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

func (r *AdminRepo) ListAllReplaySessions(ctx context.Context, status, customerID string, limit, offset int) ([]*domain.AdminReplaySession, int, error) {
	baseWhere := baseWhereTrue
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
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get replay detail: %w", err)
	}

	sourcePixelID = detail.Session.SourcePixelID
	targetPixelID = detail.Session.TargetPixelID

	var sourcePixel, targetPixel *domain.Pixel

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		p := &domain.Pixel{}
		err := r.pool.QueryRow(gCtx,
			`SELECT id, customer_id, fb_pixel_id, name, is_active, backup_pixel_id, test_event_code, created_at, updated_at
			 FROM pixels WHERE id = $1`, sourcePixelID,
		).Scan(&p.ID, &p.CustomerID, &p.FBPixelID, &p.Name, &p.IsActive, &p.BackupPixelID, &p.TestEventCode, &p.CreatedAt, &p.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		sourcePixel = p
		return nil
	})

	g.Go(func() error {
		p := &domain.Pixel{}
		err := r.pool.QueryRow(gCtx,
			`SELECT id, customer_id, fb_pixel_id, name, is_active, backup_pixel_id, test_event_code, created_at, updated_at
			 FROM pixels WHERE id = $1`, targetPixelID,
		).Scan(&p.ID, &p.CustomerID, &p.FBPixelID, &p.Name, &p.IsActive, &p.BackupPixelID, &p.TestEventCode, &p.CreatedAt, &p.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		targetPixel = p
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get replay detail sub-queries: %w", err)
	}

	detail.SourcePixel = sourcePixel
	detail.TargetPixel = targetPixel

	return detail, nil
}
