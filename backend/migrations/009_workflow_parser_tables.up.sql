CREATE TABLE workflows (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'disabled')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(repository_id, path)
);

CREATE INDEX idx_workflows_organization_id ON workflows(organization_id);
CREATE INDEX idx_workflows_repository_id ON workflows(repository_id);

CREATE TABLE workflow_revisions (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    commit_sha TEXT NOT NULL,
    raw_content_hash TEXT NOT NULL,
    parse_status TEXT NOT NULL DEFAULT 'pending' CHECK (parse_status IN ('valid', 'invalid', 'pending')),
    ir TEXT NOT NULL DEFAULT '{}',
    parsed_at TIMESTAMP,
    UNIQUE(workflow_id, commit_sha)
);

CREATE INDEX idx_workflow_revisions_workflow_id ON workflow_revisions(workflow_id);

CREATE TABLE workflow_diagnostics (
    id TEXT PRIMARY KEY,
    workflow_revision_id TEXT NOT NULL REFERENCES workflow_revisions(id) ON DELETE CASCADE,
    line INTEGER NOT NULL DEFAULT 0,
    col INTEGER NOT NULL DEFAULT 0,
    severity TEXT NOT NULL CHECK (severity IN ('error', 'warning', 'info')),
    message TEXT NOT NULL
);

CREATE INDEX idx_workflow_diagnostics_revision_id ON workflow_diagnostics(workflow_revision_id);
