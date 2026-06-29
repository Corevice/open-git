-- Migration 018_oauth_enhancements (task spec: 015_oauth_enhancements).
-- Numbering: 015=security_tables, 016=system_settings+perf_tables, 017=repository_collaborators.
-- oauth_app deletion cascades to oauth_grants and linked access_tokens (consistent revocation lifecycle).
-- token_prefix/token_last_eight store display metadata only; full token hash remains in token_hash.

ALTER TABLE access_tokens ADD COLUMN IF NOT EXISTS note TEXT;
ALTER TABLE access_tokens ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMP;
ALTER TABLE access_tokens ADD COLUMN IF NOT EXISTS token_prefix TEXT;
ALTER TABLE access_tokens ADD COLUMN IF NOT EXISTS token_last_eight TEXT;
ALTER TABLE access_tokens ADD COLUMN IF NOT EXISTS oauth_application_id TEXT REFERENCES oauth_apps(id) ON DELETE CASCADE;

ALTER TABLE oauth_apps ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE oauth_apps ADD COLUMN IF NOT EXISTS homepage_url TEXT NOT NULL DEFAULT '';

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
