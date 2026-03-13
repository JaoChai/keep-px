-- Revert per-pixel pricing migration
-- Note: data migration (plan names, addon types) cannot be perfectly reversed

-- Remove added columns
ALTER TABLE subscriptions DROP COLUMN IF EXISTS quantity;
ALTER TABLE customers DROP COLUMN IF EXISTS retention_days;
