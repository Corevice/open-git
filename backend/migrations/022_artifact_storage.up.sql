CREATE TABLE artifacts (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    repository_id TEXT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    workflow_run_id TEXT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    size_in_bytes INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','uploading','completed','failed','expired')),
    retention_days INTEGER NOT NULL DEFAULT 90
        CHECK (retention_days >= 1 AND retention_days <= 90),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP
);
CREATE UNIQUE INDEX idx_artifacts_run_name
    ON artifacts(workflow_run_id, name)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_artifacts_organization_id ON artifacts(organization_id);
CREATE INDEX idx_artifacts_expires_at ON artifacts(expires_at, status);
