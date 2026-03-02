CREATE TABLE IF NOT EXISTS notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    metadata    JSONB,
    is_read     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_notifications_customer_unread
    ON notifications(customer_id) WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_customer_created
    ON notifications(customer_id, created_at DESC);
