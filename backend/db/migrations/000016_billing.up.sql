-- Add Stripe customer ID to customers
ALTER TABLE customers ADD COLUMN IF NOT EXISTS stripe_customer_id VARCHAR(255);
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_stripe_id
    ON customers(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;

-- One-time purchases
CREATE TABLE IF NOT EXISTS purchases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    stripe_checkout_session_id VARCHAR(255) UNIQUE,
    stripe_payment_intent_id VARCHAR(255),
    pack_type VARCHAR(50) NOT NULL,
    amount_satang INT NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'thb',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_purchases_customer ON purchases(customer_id);

-- Replay credits (allocated on purchase completion)
CREATE TABLE IF NOT EXISTS replay_credits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    purchase_id UUID REFERENCES purchases(id) ON DELETE SET NULL,
    pack_type VARCHAR(50) NOT NULL,
    total_replays INT NOT NULL,
    used_replays INT NOT NULL DEFAULT 0,
    max_events_per_replay INT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_credits_customer ON replay_credits(customer_id);
CREATE INDEX IF NOT EXISTS idx_credits_active
    ON replay_credits(customer_id, expires_at)
    WHERE used_replays < total_replays OR total_replays = -1;

-- Subscriptions (recurring add-ons)
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    stripe_subscription_id VARCHAR(255) UNIQUE NOT NULL,
    stripe_price_id VARCHAR(255) NOT NULL,
    addon_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    cancel_at_period_end BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_subs_customer ON subscriptions(customer_id);

-- Monthly event usage
CREATE TABLE IF NOT EXISTS event_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    month DATE NOT NULL,
    event_count BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_customer_month ON event_usage(customer_id, month);
