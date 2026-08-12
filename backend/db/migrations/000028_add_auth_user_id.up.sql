-- Neon Auth migration: customers can be linked to a Neon Auth user.
-- auth_user_id stores the Neon Auth user id once a customer is linked; NULL
-- for customers not yet linked.
ALTER TABLE customers ADD COLUMN auth_user_id TEXT;

-- Partial unique index: many customers are not yet linked (NULL). Postgres
-- already allows duplicate NULLs in a unique index, but the WHERE clause makes
-- the intent explicit and keeps the index small.
CREATE UNIQUE INDEX idx_customers_auth_user_id ON customers (auth_user_id)
    WHERE auth_user_id IS NOT NULL;
