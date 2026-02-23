-- name: CreatePixelEvent :one
INSERT INTO pixel_events (pixel_id, event_name, event_data, user_data, source_url, event_time)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetPixelEventByID :one
SELECT * FROM pixel_events WHERE id = $1;

-- name: ListPixelEventsByPixelID :many
SELECT * FROM pixel_events
WHERE pixel_id = $1
ORDER BY event_time DESC
LIMIT $2 OFFSET $3;

-- name: CountPixelEventsByPixelID :one
SELECT COUNT(*) FROM pixel_events WHERE pixel_id = $1;

-- name: ListPixelEventsByCustomerID :many
SELECT pe.* FROM pixel_events pe
JOIN pixels p ON p.id = pe.pixel_id
WHERE p.customer_id = $1
ORDER BY pe.event_time DESC
LIMIT $2 OFFSET $3;

-- name: CountPixelEventsByCustomerID :one
SELECT COUNT(*) FROM pixel_events pe
JOIN pixels p ON p.id = pe.pixel_id
WHERE p.customer_id = $1;

-- name: MarkEventForwarded :exec
UPDATE pixel_events SET
    forwarded_to_capi = true,
    capi_response_code = $2
WHERE id = $1;

-- name: GetEventsForReplay :many
SELECT * FROM pixel_events
WHERE pixel_id = $1
  AND ($2::text[] IS NULL OR event_name = ANY($2::text[]))
  AND ($3::timestamptz IS NULL OR event_time >= $3)
  AND ($4::timestamptz IS NULL OR event_time <= $4)
ORDER BY event_time ASC;
