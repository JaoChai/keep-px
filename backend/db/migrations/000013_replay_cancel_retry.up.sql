ALTER TABLE replay_sessions ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ;
ALTER TABLE replay_sessions ADD COLUMN IF NOT EXISTS failed_batch_ranges JSONB;
