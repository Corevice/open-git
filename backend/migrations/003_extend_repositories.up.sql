ALTER TABLE repositories ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE repositories ADD COLUMN disk_path TEXT NOT NULL DEFAULT '';
ALTER TABLE repositories ADD COLUMN is_empty INTEGER NOT NULL DEFAULT 1;

CREATE TABLE ssh_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    fingerprint TEXT NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ssh_keys_user_id ON ssh_keys(user_id);

CREATE TABLE repository_collaborators (
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    permission TEXT NOT NULL CHECK (permission IN ('read', 'write', 'admin')),
    PRIMARY KEY (repository_id, user_id)
);

CREATE INDEX idx_repo_collaborators_repository_id ON repository_collaborators(repository_id);
