ALTER TABLE access_tokens ADD COLUMN IF NOT EXISTS note TEXT;
ALTER TABLE access_tokens ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMP;
ALTER TABLE access_tokens ADD COLUMN IF NOT EXISTS token_prefix TEXT;
ALTER TABLE access_tokens ADD COLUMN IF NOT EXISTS token_last_eight TEXT;
ALTER TABLE access_tokens ADD COLUMN IF NOT EXISTS oauth_application_id TEXT REFERENCES oauth_apps(id);

ALTER TABLE oauth_apps ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE oauth_apps ADD COLUMN IF NOT EXISTS homepage_url TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS oauth_grants (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    oauth_application_id TEXT NOT NULL REFERENCES oauth_apps(id),
    scopes TEXT NOT NULL DEFAULT '[]',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, oauth_application_id)
);

CREATE INDEX IF NOT EXISTS idx_oauth_grants_user_id ON oauth_grants(user_id);
