-- Admin system: add admin flag, suspension support, and credit grant audit trail

ALTER TABLE customers ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS admin_credit_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID NOT NULL REFERENCES customers(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    pack_type VARCHAR(50) NOT NULL,
    total_replays INT NOT NULL,
    max_events_per_replay INT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    reason TEXT,
    credit_id UUID REFERENCES replay_credits(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_grants_customer ON admin_credit_grants(customer_id);
CREATE INDEX IF NOT EXISTS idx_admin_grants_admin ON admin_credit_grants(admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_grants_created ON admin_credit_grants(created_at DESC);
