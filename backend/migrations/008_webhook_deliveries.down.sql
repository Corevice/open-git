DROP TABLE IF EXISTS webhook_deliveries;

ALTER TABLE webhooks DROP COLUMN content_type;
ALTER TABLE webhooks DROP COLUMN updated_at;
