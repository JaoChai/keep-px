-- name: CreateEventRule :one
INSERT INTO event_rules (pixel_id, page_url, event_name, trigger_type, css_selector, xpath, element_text, conditions, parameters, fire_once, delay_ms)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetEventRuleByID :one
SELECT * FROM event_rules WHERE id = $1;

-- name: ListEventRulesByPixelID :many
SELECT * FROM event_rules WHERE pixel_id = $1 ORDER BY created_at DESC;

-- name: ListActiveEventRulesByPixelID :many
SELECT * FROM event_rules WHERE pixel_id = $1 AND is_active = true ORDER BY created_at DESC;

-- name: UpdateEventRule :one
UPDATE event_rules SET
    page_url = COALESCE(sqlc.narg('page_url'), page_url),
    event_name = COALESCE(sqlc.narg('event_name'), event_name),
    trigger_type = COALESCE(sqlc.narg('trigger_type'), trigger_type),
    css_selector = COALESCE(sqlc.narg('css_selector'), css_selector),
    xpath = COALESCE(sqlc.narg('xpath'), xpath),
    element_text = COALESCE(sqlc.narg('element_text'), element_text),
    conditions = COALESCE(sqlc.narg('conditions'), conditions),
    parameters = COALESCE(sqlc.narg('parameters'), parameters),
    fire_once = COALESCE(sqlc.narg('fire_once'), fire_once),
    delay_ms = COALESCE(sqlc.narg('delay_ms'), delay_ms),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteEventRule :exec
DELETE FROM event_rules WHERE id = $1;
