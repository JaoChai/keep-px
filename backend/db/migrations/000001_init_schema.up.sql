-- customers (ผู้ใช้ระบบ Pixlinks)
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    api_key VARCHAR(64) UNIQUE NOT NULL,
    plan VARCHAR(50) DEFAULT 'free',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- pixels (Facebook Pixel ที่ลูกค้าเชื่อมต่อ)
CREATE TABLE pixels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    fb_pixel_id VARCHAR(50) NOT NULL,
    fb_access_token TEXT NOT NULL,
    name VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- pixel_events (ทุก event ที่เก็บไว้)
CREATE TABLE pixel_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pixel_id UUID REFERENCES pixels(id) ON DELETE CASCADE,
    event_name VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    user_data JSONB,
    source_url TEXT,
    event_time TIMESTAMPTZ NOT NULL,
    forwarded_to_capi BOOLEAN DEFAULT false,
    capi_response_code INT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- event_rules (กฎจากเครื่องมือกำหนด Event แบบ Visual)
CREATE TABLE event_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pixel_id UUID REFERENCES pixels(id) ON DELETE CASCADE,
    page_url TEXT NOT NULL,
    event_name VARCHAR(100) NOT NULL,
    trigger_type VARCHAR(50) NOT NULL,
    css_selector TEXT,
    xpath TEXT,
    element_text TEXT,
    conditions JSONB,
    parameters JSONB,
    fire_once BOOLEAN DEFAULT false,
    delay_ms INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- replay_sessions (ประวัติการ replay)
CREATE TABLE replay_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID REFERENCES customers(id),
    source_pixel_id UUID REFERENCES pixels(id),
    target_pixel_id UUID REFERENCES pixels(id),
    status VARCHAR(50) DEFAULT 'pending',
    total_events INT DEFAULT 0,
    replayed_events INT DEFAULT 0,
    failed_events INT DEFAULT 0,
    event_types TEXT[],
    date_from TIMESTAMPTZ,
    date_to TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- refresh_tokens
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_pixel_events_pixel_time ON pixel_events(pixel_id, event_time DESC);
CREATE INDEX idx_pixel_events_not_forwarded ON pixel_events(forwarded_to_capi) WHERE forwarded_to_capi = false;
CREATE INDEX idx_event_rules_pixel_active ON event_rules(pixel_id) WHERE is_active = true;
CREATE INDEX idx_customers_api_key ON customers(api_key);
CREATE INDEX idx_pixels_customer ON pixels(customer_id);
CREATE INDEX idx_replay_sessions_customer ON replay_sessions(customer_id);
