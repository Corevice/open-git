ALTER TABLE webhooks ADD COLUMN last_delivery_status TEXT NOT NULL DEFAULT '';
ALTER TABLE webhooks ADD COLUMN last_delivery_at TIMESTAMP;
