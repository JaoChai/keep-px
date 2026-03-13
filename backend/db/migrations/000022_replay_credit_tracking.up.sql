ALTER TABLE replay_sessions ADD COLUMN IF NOT EXISTS credit_id UUID REFERENCES replay_credits(id);
