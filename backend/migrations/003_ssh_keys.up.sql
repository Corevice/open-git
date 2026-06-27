CREATE TABLE ssh_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    key_type VARCHAR(32) NOT NULL,
    public_key TEXT NOT NULL,
    fingerprint VARCHAR(128) NOT NULL,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, fingerprint)
);

CREATE TABLE host_keys (
    id TEXT PRIMARY KEY,
    algorithm VARCHAR(32) NOT NULL UNIQUE,
    private_key TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
