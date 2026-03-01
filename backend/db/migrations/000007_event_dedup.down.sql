DROP INDEX IF EXISTS idx_pixel_events_pixel_event_id;
CREATE INDEX idx_pixel_events_event_id ON pixel_events(event_id);
