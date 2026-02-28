-- Replay options
ALTER TABLE replay_sessions
  ADD COLUMN IF NOT EXISTS time_mode VARCHAR(20) NOT NULL DEFAULT 'original',
  ADD COLUMN IF NOT EXISTS batch_delay_ms INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS error_message TEXT;

-- CHECK constraints (safe to re-run: DO $$ block checks existence first)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'replay_sessions_time_mode_check'
  ) THEN
    ALTER TABLE replay_sessions ADD CONSTRAINT replay_sessions_time_mode_check
      CHECK (time_mode IN ('original', 'current'));
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'replay_sessions_batch_delay_ms_check'
  ) THEN
    ALTER TABLE replay_sessions ADD CONSTRAINT replay_sessions_batch_delay_ms_check
      CHECK (batch_delay_ms >= 0);
  END IF;
END $$;

-- Backup pixel
ALTER TABLE pixels
  ADD COLUMN IF NOT EXISTS backup_pixel_id UUID REFERENCES pixels(id) ON DELETE SET NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'pixels_backup_pixel_id_check'
  ) THEN
    ALTER TABLE pixels ADD CONSTRAINT pixels_backup_pixel_id_check
      CHECK (backup_pixel_id IS DISTINCT FROM id);
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_pixels_backup_pixel_id ON pixels(backup_pixel_id)
  WHERE backup_pixel_id IS NOT NULL;
