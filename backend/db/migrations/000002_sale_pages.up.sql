CREATE TABLE sale_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    pixel_id UUID REFERENCES pixels(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    template_name VARCHAR(50) NOT NULL DEFAULT 'simple',
    content JSONB NOT NULL DEFAULT '{}',
    is_published BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sale_pages_customer ON sale_pages(customer_id);
CREATE INDEX idx_sale_pages_slug ON sale_pages(slug);
