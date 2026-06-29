ALTER TABLE workflow_runs ADD COLUMN IF NOT EXISTS head_sha TEXT;
ALTER TABLE workflow_runs ADD COLUMN IF NOT EXISTS head_branch TEXT;
ALTER TABLE workflow_runs ADD COLUMN IF NOT EXISTS run_number INTEGER NOT NULL DEFAULT 0;
ALTER TABLE workflow_runs ADD COLUMN IF NOT EXISTS run_attempt INTEGER NOT NULL DEFAULT 1;
ALTER TABLE workflow_runs ADD COLUMN IF NOT EXISTS event TEXT NOT NULL DEFAULT 'push';
ALTER TABLE workflow_runs ADD COLUMN IF NOT EXISTS triggered_by_user_id TEXT REFERENCES users(id);

CREATE TABLE IF NOT EXISTS workflow_jobs (
    id TEXT PRIMARY KEY,
    workflow_run_id TEXT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued', 'in_progress', 'completed', 'failed')),
    conclusion TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workflow_jobs_org_run ON workflow_jobs(organization_id, workflow_run_id);

CREATE TABLE IF NOT EXISTS workflow_steps (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES workflow_jobs(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    number INTEGER NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued', 'in_progress', 'completed', 'pending')),
    conclusion TEXT CHECK(conclusion IS NULL OR conclusion IN ('success', 'failure', 'cancelled', 'skipped', 'timed_out', 'action_required')),
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workflow_steps_job_id ON workflow_steps(job_id);
CREATE INDEX IF NOT EXISTS idx_workflow_steps_organization_id ON workflow_steps(organization_id);

CREATE TABLE IF NOT EXISTS job_logs (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES workflow_jobs(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    log_offset BIGINT NOT NULL,
    chunk TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_job_logs_job_offset ON job_logs(job_id, log_offset);
CREATE INDEX IF NOT EXISTS idx_job_logs_organization_id ON job_logs(organization_id, job_id);

CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    name TEXT NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    storage_key TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, storage_key)
);

CREATE INDEX IF NOT EXISTS idx_artifacts_run_id ON artifacts(run_id);

CREATE TABLE IF NOT EXISTS action_secrets (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT REFERENCES repositories(id),
    name TEXT NOT NULL,
    encrypted_value TEXT NOT NULL,
    encryption_algorithm TEXT NOT NULL DEFAULT 'aes-256-gcm',
    encryption_key_version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_action_secrets_organization_id ON action_secrets(organization_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_action_secrets_scope_name ON action_secrets(organization_id, COALESCE(repository_id, ''), name);

CREATE TRIGGER IF NOT EXISTS trg_action_secrets_updated_at
AFTER UPDATE ON action_secrets
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE action_secrets
    SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;
