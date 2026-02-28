CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pixel_events_pixel_created
ON pixel_events (pixel_id, created_at ASC);
