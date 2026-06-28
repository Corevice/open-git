CREATE TABLE runners (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    name TEXT NOT NULL,
    labels TEXT NOT NULL DEFAULT '[]',
    os TEXT NOT NULL DEFAULT '',
    arch TEXT NOT NULL DEFAULT '',
    runner_type TEXT NOT NULL CHECK(runner_type IN ('act','official')),
    status TEXT NOT NULL DEFAULT 'offline' CHECK(status IN ('online','offline','busy')),
    last_seen_at TIMESTAMPTZ,
    ephemeral BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_runners_organization_id ON runners(organization_id);

CREATE TABLE runner_registration_tokens (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);

CREATE INDEX idx_runner_registration_tokens_organization_id ON runner_registration_tokens(organization_id);

CREATE TABLE workflow_jobs (
    id TEXT PRIMARY KEY,
    workflow_run_id TEXT REFERENCES workflow_runs(id),
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    name TEXT NOT NULL DEFAULT '',
    runs_on TEXT NOT NULL DEFAULT '[]',
    assigned_runner_id TEXT REFERENCES runners(id),
    acquire_lock_version INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','in_progress','completed','failed','cancelled')),
    conclusion TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    timeout_minutes INTEGER NOT NULL DEFAULT 360,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_workflow_jobs_organization_id ON workflow_jobs(organization_id);
CREATE INDEX idx_workflow_jobs_status ON workflow_jobs(status);
CREATE INDEX idx_workflow_jobs_assigned_runner_id ON workflow_jobs(assigned_runner_id);

CREATE TABLE runner_audit_log (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    actor_id TEXT,
    action TEXT NOT NULL,
    target_runner_id TEXT REFERENCES runners(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_runner_audit_log_organization_id ON runner_audit_log(organization_id);
