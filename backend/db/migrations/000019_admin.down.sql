DROP TABLE IF EXISTS admin_credit_grants;
ALTER TABLE customers DROP COLUMN IF EXISTS suspended_at;
ALTER TABLE customers DROP COLUMN IF EXISTS is_admin;
