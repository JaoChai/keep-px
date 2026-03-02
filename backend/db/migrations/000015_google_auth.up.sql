ALTER TABLE customers ALTER COLUMN password_hash DROP NOT NULL;

ALTER TABLE customers ADD COLUMN IF NOT EXISTS google_id VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_google_id ON customers(google_id) WHERE google_id IS NOT NULL;
