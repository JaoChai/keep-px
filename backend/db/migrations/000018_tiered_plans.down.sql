-- Revert sandbox back to free
UPDATE customers SET plan = 'free' WHERE plan = 'sandbox';
ALTER TABLE customers ALTER COLUMN plan SET DEFAULT 'free';
