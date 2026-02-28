ALTER TABLE replay_sessions
  DROP COLUMN IF EXISTS time_mode,
  DROP COLUMN IF EXISTS batch_delay_ms,
  DROP COLUMN IF EXISTS error_message;

DROP INDEX IF EXISTS idx_pixels_backup_pixel_id;

ALTER TABLE pixels
  DROP COLUMN IF EXISTS backup_pixel_id;
