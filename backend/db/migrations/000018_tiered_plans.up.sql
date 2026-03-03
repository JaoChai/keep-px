-- Migrate existing "free" customers to "sandbox"
UPDATE customers SET plan = 'sandbox' WHERE plan = 'free';
ALTER TABLE customers ALTER COLUMN plan SET DEFAULT 'sandbox';

-- Cancel old retention add-ons (no longer valid under tiered plans)
UPDATE subscriptions SET status = 'canceled', updated_at = NOW()
  WHERE addon_type IN ('retention_180', 'retention_365') AND status = 'active';
