-- name: CreateCustomer :one
INSERT INTO customers (email, password_hash, name, api_key, plan)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetCustomerByID :one
SELECT * FROM customers WHERE id = $1;

-- name: GetCustomerByEmail :one
SELECT * FROM customers WHERE email = $1;

-- name: GetCustomerByAPIKey :one
SELECT * FROM customers WHERE api_key = $1;

-- name: UpdateCustomer :one
UPDATE customers SET
    email = COALESCE(sqlc.narg('email'), email),
    name = COALESCE(sqlc.narg('name'), name),
    plan = COALESCE(sqlc.narg('plan'), plan),
    updated_at = NOW()
WHERE id = $1
RETURNING *;
