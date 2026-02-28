CREATE TABLE IF NOT EXISTS sale_page_pixels (
    sale_page_id UUID NOT NULL REFERENCES sale_pages(id) ON DELETE CASCADE,
    pixel_id UUID NOT NULL REFERENCES pixels(id) ON DELETE CASCADE,
    position SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (sale_page_id, pixel_id)
);
CREATE INDEX IF NOT EXISTS idx_sale_page_pixels_pixel ON sale_page_pixels(pixel_id);

-- Migrate existing data
INSERT INTO sale_page_pixels (sale_page_id, pixel_id, position)
SELECT id, pixel_id, 0 FROM sale_pages WHERE pixel_id IS NOT NULL
ON CONFLICT DO NOTHING;

-- Drop old column
ALTER TABLE sale_pages DROP COLUMN IF EXISTS pixel_id;
