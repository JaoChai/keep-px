ALTER TABLE sale_pages ADD COLUMN IF NOT EXISTS pixel_id UUID REFERENCES pixels(id) ON DELETE SET NULL;
UPDATE sale_pages sp SET pixel_id = (
    SELECT pixel_id FROM sale_page_pixels spp WHERE spp.sale_page_id = sp.id ORDER BY position LIMIT 1
);
DROP TABLE IF EXISTS sale_page_pixels;
