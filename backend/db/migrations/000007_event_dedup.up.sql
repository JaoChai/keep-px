-- Drop existing non-unique index on event_id
DROP INDEX IF EXISTS idx_pixel_events_event_id;

-- Partial unique index: prevent duplicate (pixel_id, event_id) pairs
-- NULLs in event_id are excluded (old rows pre-migration 000004)
CREATE UNIQUE INDEX idx_pixel_events_pixel_event_id
  ON pixel_events (pixel_id, event_id)
  WHERE event_id IS NOT NULL;
