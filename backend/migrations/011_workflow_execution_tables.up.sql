ALTER TABLE workflow_runs ADD COLUMN IF NOT EXISTS head_sha TEXT, ADD COLUMN IF NOT EXISTS head_branch TEXT, ADD COLUMN IF NOT EXISTS run_number INTEGER NOT NULL DEFAULT 0, ADD COLUMN IF NOT EXISTS run_attempt INTEGER NOT NULL DEFAULT 1, ADD COLUMN IF NOT EXISTS event TEXT NOT NULL DEFAULT 'push', ADD COLUMN IF NOT EXISTS triggered_by_user_id TEXT REFERENCES users(id);

CREATE TABLE workflow_jobs (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    conclusion TEXT,
    needs TEXT NOT NULL DEFAULT '[]',
    matrix_context TEXT NOT NULL DEFAULT '{}',
    runner_label TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_workflow_jobs_run_id ON workflow_jobs(run_id);
CREATE INDEX idx_workflow_jobs_organization_id ON workflow_jobs(organization_id);

CREATE TABLE workflow_steps (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES workflow_jobs(id) ON DELETE CASCADE,
    number INTEGER NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    conclusion TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX idx_workflow_steps_job_id ON workflow_steps(job_id);

CREATE TABLE job_logs (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES workflow_jobs(id) ON DELETE CASCADE,
    log_offset BIGINT NOT NULL,
    chunk TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_job_logs_job_offset ON job_logs(job_id, log_offset);

CREATE TABLE artifacts (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    name TEXT NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    storage_key TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_artifacts_run_id ON artifacts(run_id);

CREATE TABLE action_secrets (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT REFERENCES repositories(id),
    name TEXT NOT NULL,
    encrypted_value TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, COALESCE(repository_id, ''), name)
);

CREATE INDEX idx_action_secrets_organization_id ON action_secrets(organization_id);
