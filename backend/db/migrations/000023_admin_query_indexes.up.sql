-- purchases: used in GetPlatformStats (revenue queries), GetRevenueChart
CREATE INDEX IF NOT EXISTS idx_purchases_status_completed
  ON purchases(completed_at DESC) WHERE status = 'completed';

-- replay_sessions: used in GetPlatformStats, ListAllReplaySessions
CREATE INDEX IF NOT EXISTS idx_replay_sessions_status
  ON replay_sessions(status);

-- sale_page_pixels: used in ListAllSalePages JOIN aggregation
CREATE INDEX IF NOT EXISTS idx_sale_page_pixels_sale_page
  ON sale_page_pixels(sale_page_id);

-- pixel_events: plain pixel_id index for COUNT(*) aggregation
-- idx_pixel_events_pixel_time(pixel_id, event_time DESC) exists but
-- a single-column index is more efficient for COUNT(*)
CREATE INDEX IF NOT EXISTS idx_pixel_events_pixel_count
  ON pixel_events(pixel_id);
