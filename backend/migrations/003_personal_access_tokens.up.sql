CREATE TABLE personal_access_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    token_last_eight TEXT NOT NULL DEFAULT '',
    scopes TEXT NOT NULL DEFAULT '[]',
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_pat_user_id ON personal_access_tokens(user_id);
CREATE UNIQUE INDEX idx_pat_token_hash ON personal_access_tokens(token_hash);
CREATE UNIQUE INDEX idx_pat_user_name ON personal_access_tokens(user_id, name);
