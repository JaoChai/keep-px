-- name: CreateReplaySession :one
INSERT INTO replay_sessions (customer_id, source_pixel_id, target_pixel_id, total_events, event_types, date_from, date_to)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetReplaySessionByID :one
SELECT * FROM replay_sessions WHERE id = $1;

-- name: ListReplaySessionsByCustomerID :many
SELECT * FROM replay_sessions WHERE customer_id = $1 ORDER BY created_at DESC;

-- name: UpdateReplayProgress :exec
UPDATE replay_sessions SET
    replayed_events = $2,
    failed_events = $3
WHERE id = $1;

-- name: UpdateReplayStatus :exec
UPDATE replay_sessions SET
    status = $2,
    started_at = CASE WHEN $2 = 'running' AND started_at IS NULL THEN NOW() ELSE started_at END,
    completed_at = CASE WHEN $2 IN ('completed', 'failed') THEN NOW() ELSE completed_at END
WHERE id = $1;
