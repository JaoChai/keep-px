DROP INDEX IF EXISTS idx_customers_google_id;

ALTER TABLE customers DROP COLUMN IF EXISTS google_id;

ALTER TABLE customers ALTER COLUMN password_hash SET NOT NULL;
