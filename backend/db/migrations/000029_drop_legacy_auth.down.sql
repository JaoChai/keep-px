-- คืน google_id ตามนิยามจริงใน 000015_google_auth.up.sql
-- (VARCHAR(255) ไม่มี UNIQUE constraint ที่ column — ใช้ partial unique index แทน)
ALTER TABLE customers ADD COLUMN IF NOT EXISTS google_id VARCHAR(255);
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_google_id ON customers(google_id) WHERE google_id IS NOT NULL;

-- คืน refresh_tokens ตามนิยามใน 000001_init_schema.up.sql
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
