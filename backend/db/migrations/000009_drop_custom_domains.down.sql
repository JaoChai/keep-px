CREATE TABLE IF NOT EXISTS custom_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    sale_page_id UUID NOT NULL REFERENCES sale_pages(id) ON DELETE CASCADE,
    domain VARCHAR(255) NOT NULL UNIQUE,
    cf_hostname_id VARCHAR(255),
    verification_token VARCHAR(255) NOT NULL,
    dns_verified BOOLEAN DEFAULT FALSE,
    ssl_active BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_domains_customer ON custom_domains(customer_id);
CREATE INDEX IF NOT EXISTS idx_custom_domains_domain ON custom_domains(domain);
CREATE INDEX IF NOT EXISTS idx_custom_domains_sale_page ON custom_domains(sale_page_id);
