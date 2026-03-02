package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type ReplaySessionRepo struct {
	pool *pgxpool.Pool
}

func NewReplaySessionRepo(pool *pgxpool.Pool) *ReplaySessionRepo {
	return &ReplaySessionRepo{pool: pool}
}

const replaySessionColumns = `id, customer_id, source_pixel_id, target_pixel_id, status, total_events, replayed_events, failed_events, event_types, date_from, date_to, time_mode, batch_delay_ms, error_message, started_at, completed_at, cancelled_at, failed_batch_ranges, created_at`

func scanReplaySession(row pgx.Row) (*domain.ReplaySession, error) {
	s := &domain.ReplaySession{}
	err := row.Scan(&s.ID, &s.CustomerID, &s.SourcePixelID, &s.TargetPixelID, &s.Status, &s.TotalEvents, &s.ReplayedEvents, &s.FailedEvents, &s.EventTypes, &s.DateFrom, &s.DateTo, &s.TimeMode, &s.BatchDelayMs, &s.ErrorMessage, &s.StartedAt, &s.CompletedAt, &s.CancelledAt, &s.FailedBatchRanges, &s.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func (r *ReplaySessionRepo) Create(ctx context.Context, session *domain.ReplaySession) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO replay_sessions (customer_id, source_pixel_id, target_pixel_id, total_events, event_types, date_from, date_to, time_mode, batch_delay_ms)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, status, replayed_events, failed_events, created_at`,
		session.CustomerID, session.SourcePixelID, session.TargetPixelID, session.TotalEvents, session.EventTypes, session.DateFrom, session.DateTo, session.TimeMode, session.BatchDelayMs,
	).Scan(&session.ID, &session.Status, &session.ReplayedEvents, &session.FailedEvents, &session.CreatedAt)
}

func (r *ReplaySessionRepo) GetByID(ctx context.Context, id string) (*domain.ReplaySession, error) {
	return scanReplaySession(r.pool.QueryRow(ctx,
		`SELECT `+replaySessionColumns+` FROM replay_sessions WHERE id = $1`, id,
	))
}

func (r *ReplaySessionRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.ReplaySession, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+replaySessionColumns+` FROM replay_sessions WHERE customer_id = $1 ORDER BY created_at DESC`, customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.ReplaySession
	for rows.Next() {
		s := &domain.ReplaySession{}
		if err := rows.Scan(&s.ID, &s.CustomerID, &s.SourcePixelID, &s.TargetPixelID, &s.Status, &s.TotalEvents, &s.ReplayedEvents, &s.FailedEvents, &s.EventTypes, &s.DateFrom, &s.DateTo, &s.TimeMode, &s.BatchDelayMs, &s.ErrorMessage, &s.StartedAt, &s.CompletedAt, &s.CancelledAt, &s.FailedBatchRanges, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *ReplaySessionRepo) UpdateProgress(ctx context.Context, id string, replayed, failed int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE replay_sessions SET replayed_events = $2, failed_events = $3 WHERE id = $1`,
		id, replayed, failed,
	)
	return err
}

func (r *ReplaySessionRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE replay_sessions SET
			status = $2,
			started_at = CASE WHEN $2 = 'running' AND started_at IS NULL THEN NOW() ELSE started_at END,
			completed_at = CASE WHEN $2 IN ('completed', 'failed', 'cancelled') THEN NOW() ELSE completed_at END,
			cancelled_at = CASE WHEN $2 = 'cancelled' THEN NOW() ELSE cancelled_at END
		 WHERE id = $1`,
		id, status,
	)
	return err
}

func (r *ReplaySessionRepo) UpdateStatusWithError(ctx context.Context, id string, status string, errorMsg string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE replay_sessions SET
			status = $2,
			error_message = $3,
			started_at = CASE WHEN $2 = 'running' AND started_at IS NULL THEN NOW() ELSE started_at END,
			completed_at = CASE WHEN $2 IN ('completed', 'failed', 'cancelled') THEN NOW() ELSE completed_at END,
			cancelled_at = CASE WHEN $2 = 'cancelled' THEN NOW() ELSE cancelled_at END
		 WHERE id = $1`,
		id, status, errorMsg,
	)
	return err
}

func (r *ReplaySessionRepo) UpdateTotalEvents(ctx context.Context, id string, total int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE replay_sessions SET total_events = $2 WHERE id = $1`,
		id, total,
	)
	return err
}

func (r *ReplaySessionRepo) GetStatus(ctx context.Context, id string) (string, error) {
	var status string
	err := r.pool.QueryRow(ctx,
		`SELECT status FROM replay_sessions WHERE id = $1`, id,
	).Scan(&status)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return status, err
}

func (r *ReplaySessionRepo) CancelSession(ctx context.Context, id string) (*domain.ReplaySession, error) {
	return scanReplaySession(r.pool.QueryRow(ctx,
		`UPDATE replay_sessions SET
			status = 'cancelled',
			cancelled_at = NOW(),
			completed_at = NOW()
		 WHERE id = $1 AND status IN ('pending', 'running')
		 RETURNING `+replaySessionColumns,
		id,
	))
}

func (r *ReplaySessionRepo) UpdateFailedBatches(ctx context.Context, id string, failedBatchRanges []byte) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE replay_sessions SET failed_batch_ranges = $2 WHERE id = $1`,
		id, failedBatchRanges,
	)
	return err
}

func (r *ReplaySessionRepo) RecoverOrphanedSessions(ctx context.Context) (int, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE replay_sessions
		 SET status = 'failed',
		     error_message = CASE
		         WHEN status = 'running' THEN 'server restarted during replay'
		         ELSE 'server restarted before replay started'
		     END,
		     completed_at = NOW()
		 WHERE status IN ('running', 'pending')`)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
