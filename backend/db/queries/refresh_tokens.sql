-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (customer_id, token_hash, expires_at)
VALUES ($1, $2, $3);

-- name: GetRefreshToken :one
SELECT customer_id, expires_at FROM refresh_tokens
WHERE token_hash = $1 AND expires_at > NOW();

-- name: DeleteRefreshTokensByCustomerID :exec
DELETE FROM refresh_tokens WHERE customer_id = $1;

-- name: DeleteRefreshTokenByHash :exec
DELETE FROM refresh_tokens WHERE token_hash = $1;
