ALTER TABLE pixels ADD COLUMN status VARCHAR(20) DEFAULT 'active';
UPDATE pixels SET status = CASE WHEN is_active THEN 'active' ELSE 'paused' END;
