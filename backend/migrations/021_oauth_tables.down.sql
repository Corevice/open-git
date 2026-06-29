DROP TABLE IF EXISTS oauth_authorizations;
DROP TABLE IF EXISTS oauth_access_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;

-- SQLite does not support DROP COLUMN before version 3.35.
-- Recreate oauth_apps without the columns added in this migration
-- (owner_type, organization_id, updated_at). The name/homepage_url columns
-- were added in 019_oauth_enhancements and must be preserved here.
CREATE TABLE oauth_apps_rollback AS
SELECT id, owner_id, client_id, client_secret_hash, redirect_uris, name, homepage_url, created_at
FROM oauth_apps;

DROP TABLE oauth_apps;

CREATE TABLE oauth_apps (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users(id),
    client_id TEXT NOT NULL UNIQUE,
    client_secret_hash TEXT NOT NULL,
    redirect_uris TEXT NOT NULL DEFAULT '[]',
    name TEXT NOT NULL DEFAULT '',
    homepage_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO oauth_apps (id, owner_id, client_id, client_secret_hash, redirect_uris, name, homepage_url, created_at)
SELECT id, owner_id, client_id, client_secret_hash, redirect_uris, name, homepage_url, created_at FROM oauth_apps_rollback;
DROP TABLE oauth_apps_rollback;
