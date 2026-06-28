ALTER TABLE oauth_apps ADD COLUMN name TEXT NOT NULL DEFAULT '';
ALTER TABLE oauth_apps ADD COLUMN homepage_url TEXT NOT NULL DEFAULT '';
ALTER TABLE oauth_apps ADD COLUMN owner_type TEXT NOT NULL DEFAULT 'user';
ALTER TABLE oauth_apps ADD COLUMN owner_user_id TEXT REFERENCES users(id);
ALTER TABLE oauth_apps ADD COLUMN organization_id TEXT REFERENCES organizations(id);
ALTER TABLE oauth_apps ADD COLUMN updated_at TIMESTAMP;

CREATE TABLE oauth_authorization_codes (
    id TEXT PRIMARY KEY,
    code_hash TEXT NOT NULL UNIQUE,
    oauth_app_id TEXT NOT NULL REFERENCES oauth_apps(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    redirect_uri TEXT NOT NULL,
    scopes TEXT NOT NULL DEFAULT '[]',
    expires_at TIMESTAMP NOT NULL,
    consumed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_oauth_auth_codes_app ON oauth_authorization_codes(oauth_app_id);

CREATE TABLE oauth_access_tokens (
    id TEXT PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    oauth_app_id TEXT NOT NULL REFERENCES oauth_apps(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    scopes TEXT NOT NULL DEFAULT '[]',
    revoked_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_oauth_access_tokens_token_hash ON oauth_access_tokens(token_hash);
CREATE INDEX idx_oauth_access_tokens_app_user ON oauth_access_tokens(oauth_app_id, user_id);

CREATE TABLE oauth_authorizations (
    id TEXT PRIMARY KEY,
    oauth_app_id TEXT NOT NULL REFERENCES oauth_apps(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    granted_scopes TEXT NOT NULL DEFAULT '[]',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(oauth_app_id, user_id)
);
