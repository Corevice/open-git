CREATE TABLE action_secrets (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT REFERENCES repositories(id),
    name TEXT NOT NULL,
    encrypted_value BLOB NOT NULL,
    key_id TEXT NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'all' CHECK (visibility IN ('all', 'private', 'selected')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_action_secrets_org_repo_name ON action_secrets(organization_id, COALESCE(repository_id, ''), name);
CREATE INDEX idx_action_secrets_org_repo ON action_secrets(organization_id, repository_id);

CREATE TABLE action_secret_repositories (
    secret_id TEXT NOT NULL REFERENCES action_secrets(id) ON DELETE CASCADE,
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    PRIMARY KEY (secret_id, repository_id)
);
