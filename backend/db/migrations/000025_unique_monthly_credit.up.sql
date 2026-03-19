-- Prevent duplicate monthly credit grants from concurrent Stripe webhooks.
-- Uses a unique partial index on (customer_id, pack_type, expires_at)
-- scoped to replay_monthly credits only.
CREATE UNIQUE INDEX IF NOT EXISTS idx_replay_credits_monthly_dedup
    ON replay_credits (customer_id, pack_type, expires_at)
    WHERE pack_type = 'replay_monthly';
