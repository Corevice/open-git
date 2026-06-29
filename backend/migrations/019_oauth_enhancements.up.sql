-- oauth enhancements: access token display metadata + oauth grants.
-- (note/last_used_at already added in 003_user_profile_fields.)
-- Single ADD COLUMN per statement (no IF NOT EXISTS) for Postgres+SQLite compatibility.

ALTER TABLE access_tokens ADD COLUMN token_prefix TEXT;
ALTER TABLE access_tokens ADD COLUMN token_last_eight TEXT;
ALTER TABLE access_tokens ADD COLUMN oauth_application_id TEXT REFERENCES oauth_apps(id) ON DELETE CASCADE;

ALTER TABLE oauth_apps ADD COLUMN name TEXT NOT NULL DEFAULT '';
ALTER TABLE oauth_apps ADD COLUMN homepage_url TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS oauth_grants (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    oauth_application_id TEXT NOT NULL REFERENCES oauth_apps(id) ON DELETE CASCADE,
    scopes TEXT NOT NULL DEFAULT '[]' CHECK (length(scopes) >= 2 AND substr(scopes, 1, 1) = '[' AND substr(scopes, length(scopes), 1) = ']'),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, oauth_application_id)
);

CREATE INDEX IF NOT EXISTS idx_oauth_grants_user_id ON oauth_grants(user_id);
