ALTER TABLE webhooks ADD COLUMN content_type TEXT NOT NULL DEFAULT 'json';
ALTER TABLE webhooks ADD COLUMN updated_at TIMESTAMP;

CREATE TABLE webhook_deliveries (
    id TEXT PRIMARY KEY,
    webhook_id TEXT NOT NULL REFERENCES webhooks(id),
    organization_id TEXT NOT NULL,
    event TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    status_code INTEGER,
    request_headers TEXT NOT NULL DEFAULT '{}',
    request_body TEXT NOT NULL DEFAULT '',
    response_headers TEXT,
    response_body TEXT,
    duration_ms INTEGER,
    attempt INTEGER NOT NULL DEFAULT 1,
    redelivery INTEGER NOT NULL DEFAULT 0,
    parent_delivery_id TEXT REFERENCES webhook_deliveries(id),
    delivered_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_webhook_deliveries_webhook_created ON webhook_deliveries(webhook_id, created_at DESC);
CREATE INDEX idx_webhook_deliveries_org ON webhook_deliveries(organization_id);
