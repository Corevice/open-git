-- SQLite 3.35+ supports DROP COLUMN; project tests use go-sqlite3 with DROP COLUMN (see 002_webhook_delivery_status.down.sql).
ALTER TABLE access_tokens DROP COLUMN last_used_at;
ALTER TABLE access_tokens DROP COLUMN note;

ALTER TABLE users DROP COLUMN updated_at;
ALTER TABLE users DROP COLUMN avatar_url;
ALTER TABLE users DROP COLUMN bio;
ALTER TABLE users DROP COLUMN name;
