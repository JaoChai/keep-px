DROP INDEX IF EXISTS idx_customers_auth_user_id;
ALTER TABLE customers DROP COLUMN IF EXISTS auth_user_id;
