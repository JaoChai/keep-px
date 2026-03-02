DROP TABLE IF EXISTS event_usage;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS replay_credits;
DROP TABLE IF EXISTS purchases;
ALTER TABLE customers DROP COLUMN IF EXISTS stripe_customer_id;
