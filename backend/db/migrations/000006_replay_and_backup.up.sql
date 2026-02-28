-- Replay options
ALTER TABLE replay_sessions
  ADD COLUMN time_mode VARCHAR(20) NOT NULL DEFAULT 'original'
    CHECK (time_mode IN ('original', 'current')),
  ADD COLUMN batch_delay_ms INT NOT NULL DEFAULT 0
    CHECK (batch_delay_ms >= 0),
  ADD COLUMN error_message TEXT;

-- Backup pixel
ALTER TABLE pixels
  ADD COLUMN backup_pixel_id UUID REFERENCES pixels(id) ON DELETE SET NULL
    CHECK (backup_pixel_id IS DISTINCT FROM id);

CREATE INDEX idx_pixels_backup_pixel_id ON pixels(backup_pixel_id)
  WHERE backup_pixel_id IS NOT NULL;
