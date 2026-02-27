ALTER TABLE pixel_events
  ADD COLUMN event_id VARCHAR(36),
  ADD COLUMN client_ip TEXT,
  ADD COLUMN client_user_agent TEXT;

CREATE INDEX idx_pixel_events_event_id ON pixel_events(event_id);
