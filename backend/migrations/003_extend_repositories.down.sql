DROP TABLE IF EXISTS repository_collaborators;
DROP TABLE IF EXISTS ssh_keys;

CREATE TABLE repositories_old (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    owner_id TEXT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'internal', 'public')),
    default_branch TEXT NOT NULL DEFAULT 'main',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(owner_id, name)
);

INSERT INTO repositories_old (id, organization_id, owner_id, name, visibility, default_branch, created_at)
SELECT id, organization_id, owner_id, name, visibility, default_branch, created_at
FROM repositories;

DROP TABLE repositories;

ALTER TABLE repositories_old RENAME TO repositories;

CREATE INDEX idx_repositories_organization_id ON repositories(organization_id);
