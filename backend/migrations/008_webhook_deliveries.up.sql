ALTER TABLE webhooks ADD COLUMN content_type TEXT NOT NULL DEFAULT 'json';
ALTER TABLE webhooks ADD COLUMN updated_at TIMESTAMPTZ;

CREATE TABLE webhook_deliveries (
    id UUID PRIMARY KEY,
    webhook_id BIGINT NOT NULL REFERENCES webhooks(id),
    organization_id TEXT NOT NULL,
    event TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    status_code INT,
    request_headers JSONB NOT NULL DEFAULT '{}',
    request_body TEXT NOT NULL DEFAULT '',
    response_headers JSONB,
    response_body TEXT,
    duration_ms INT,
    attempt INT NOT NULL DEFAULT 1,
    redelivery BOOLEAN NOT NULL DEFAULT FALSE,
    parent_delivery_id UUID REFERENCES webhook_deliveries(id),
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_webhook_deliveries_webhook_created ON webhook_deliveries(webhook_id, created_at DESC);
CREATE INDEX idx_webhook_deliveries_org ON webhook_deliveries(organization_id);
