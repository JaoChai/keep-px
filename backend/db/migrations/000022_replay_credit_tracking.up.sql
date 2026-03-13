ALTER TABLE replay_sessions ADD COLUMN credit_id UUID REFERENCES replay_credits(id);
