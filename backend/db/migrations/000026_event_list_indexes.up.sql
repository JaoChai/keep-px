-- Composite indexes to speed up the event list page queries (#138).
-- ListByCustomerID JOINs pixels→pixel_events and filters by event_name + event_time.
-- The existing idx_pixel_events_pixel_time covers (pixel_id, event_time DESC) but not event_name.
-- This index covers filtered-by-event-name queries via the JOIN path.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pixel_events_pixel_event_time
    ON pixel_events (pixel_id, event_name, event_time DESC);

-- ListRecentByCustomerID filters on created_at > $since with an optional pixel_id filter.
-- The existing idx_pixel_events_pixel_created covers (pixel_id, created_at ASC) but
-- the query's leading filter is created_at, not pixel_id.
-- This index supports the polling query pattern: created_at > ? ORDER BY created_at ASC.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pixel_events_created_asc
    ON pixel_events (created_at ASC);
