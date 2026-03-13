-- Per-pixel pricing migration: simplify billing from 4 plans × 3 addons to per-pixel slots
-- Each pixel slot = 1 pixel + 1 sale page + 100K events/mo + 180 day retention

-- Add quantity column to subscriptions for quantity-based pixel slots
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS quantity INT NOT NULL DEFAULT 1;

-- Add retention_days to customers (explicit per-customer override)
ALTER TABLE customers ADD COLUMN IF NOT EXISTS retention_days INT NOT NULL DEFAULT 7;

-- Migrate existing paid customers to new "paid" plan with 180-day retention
UPDATE customers SET plan = 'paid', retention_days = 180
  WHERE plan IN ('launch', 'shield', 'vault');

-- Migrate plan subscriptions → pixel_slots with correct quantity mapping
UPDATE subscriptions SET addon_type = 'pixel_slots', quantity = 10
  WHERE addon_type = 'plan_launch' AND status = 'active';
UPDATE subscriptions SET addon_type = 'pixel_slots', quantity = 25
  WHERE addon_type = 'plan_shield' AND status = 'active';
UPDATE subscriptions SET addon_type = 'pixel_slots', quantity = 50
  WHERE addon_type = 'plan_vault' AND status = 'active';

-- Cancel old addon subscriptions (no longer supported)
UPDATE subscriptions SET status = 'canceled', updated_at = NOW()
  WHERE addon_type IN ('events_1m', 'sale_pages_10', 'pixels_10') AND status = 'active';

-- Sandbox customers keep defaults (7-day retention)
UPDATE customers SET retention_days = 7 WHERE plan = 'sandbox';
