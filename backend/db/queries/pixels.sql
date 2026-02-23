-- name: CreatePixel :one
INSERT INTO pixels (customer_id, fb_pixel_id, fb_access_token, name)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPixelByID :one
SELECT * FROM pixels WHERE id = $1;

-- name: ListPixelsByCustomerID :many
SELECT * FROM pixels WHERE customer_id = $1 ORDER BY created_at DESC;

-- name: UpdatePixel :one
UPDATE pixels SET
    fb_pixel_id = COALESCE(sqlc.narg('fb_pixel_id'), fb_pixel_id),
    fb_access_token = COALESCE(sqlc.narg('fb_access_token'), fb_access_token),
    name = COALESCE(sqlc.narg('name'), name),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeletePixel :exec
DELETE FROM pixels WHERE id = $1;
