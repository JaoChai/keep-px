package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type ReplayCreditRepo struct {
	pool *pgxpool.Pool
}

func NewReplayCreditRepo(pool *pgxpool.Pool) *ReplayCreditRepo {
	return &ReplayCreditRepo{pool: pool}
}

func scanReplayCredit(row pgx.Row) (*domain.ReplayCredit, error) {
	c := &domain.ReplayCredit{}
	err := row.Scan(
		&c.ID, &c.CustomerID, &c.PurchaseID, &c.PackType,
		&c.TotalReplays, &c.UsedReplays, &c.MaxEventsPerReplay,
		&c.ExpiresAt, &c.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

const replayCreditCols = `id, customer_id, purchase_id, pack_type,
	total_replays, used_replays, max_events_per_replay, expires_at, created_at`

func (r *ReplayCreditRepo) Create(ctx context.Context, c *domain.ReplayCredit) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO replay_credits (customer_id, purchase_id, pack_type,
			total_replays, used_replays, max_events_per_replay, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at`,
		c.CustomerID, c.PurchaseID, c.PackType,
		c.TotalReplays, c.UsedReplays, c.MaxEventsPerReplay, c.ExpiresAt,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *ReplayCreditRepo) GetByID(ctx context.Context, id string) (*domain.ReplayCredit, error) {
	return scanReplayCredit(r.pool.QueryRow(ctx,
		`SELECT `+replayCreditCols+` FROM replay_credits WHERE id = $1`, id,
	))
}

func (r *ReplayCreditRepo) GetActiveByCustomerID(ctx context.Context, customerID string) ([]*domain.ReplayCredit, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+replayCreditCols+`
		 FROM replay_credits
		 WHERE customer_id = $1
		   AND (total_replays = -1 OR used_replays < total_replays)
		   AND expires_at > NOW()
		 ORDER BY expires_at ASC`,
		customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credits []*domain.ReplayCredit
	for rows.Next() {
		c := &domain.ReplayCredit{}
		if err := rows.Scan(
			&c.ID, &c.CustomerID, &c.PurchaseID, &c.PackType,
			&c.TotalReplays, &c.UsedReplays, &c.MaxEventsPerReplay,
			&c.ExpiresAt, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		credits = append(credits, c)
	}
	return credits, rows.Err()
}

func (r *ReplayCreditRepo) IncrementUsed(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE replay_credits SET used_replays = used_replays + 1
		 WHERE id = $1 AND (total_replays = -1 OR used_replays < total_replays)`,
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ConsumeOneCredit atomically selects the earliest-expiring active credit for a
// customer and increments its used_replays counter. It uses SELECT ... FOR UPDATE
// SKIP LOCKED to prevent race conditions when multiple concurrent requests try
// to consume credits simultaneously.
// Returns nil, nil if no credits are available.
func (r *ReplayCreditRepo) ConsumeOneCredit(ctx context.Context, customerID string, maxEventCount int) (*domain.ReplayCredit, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var credit domain.ReplayCredit
	err = tx.QueryRow(ctx, `
		SELECT `+replayCreditCols+`
		FROM replay_credits
		WHERE customer_id = $1
		  AND (total_replays = -1 OR used_replays < total_replays)
		  AND (max_events_per_replay = 0 OR max_events_per_replay >= $2)
		  AND expires_at > NOW()
		ORDER BY expires_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`, customerID, maxEventCount).Scan(
		&credit.ID, &credit.CustomerID, &credit.PurchaseID, &credit.PackType,
		&credit.TotalReplays, &credit.UsedReplays, &credit.MaxEventsPerReplay,
		&credit.ExpiresAt, &credit.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE replay_credits SET used_replays = used_replays + 1
		WHERE id = $1 AND (total_replays = -1 OR used_replays < total_replays)`, credit.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	credit.UsedReplays++
	return &credit, nil
}
