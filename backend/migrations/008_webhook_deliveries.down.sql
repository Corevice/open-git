DROP TABLE IF EXISTS webhook_deliveries;

ALTER TABLE webhooks DROP COLUMN IF EXISTS content_type;
ALTER TABLE webhooks DROP COLUMN IF EXISTS updated_at;
