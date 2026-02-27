DROP INDEX IF EXISTS idx_pixel_events_event_id;
ALTER TABLE pixel_events
  DROP COLUMN IF EXISTS event_id,
  DROP COLUMN IF EXISTS client_ip,
  DROP COLUMN IF EXISTS client_user_agent;
