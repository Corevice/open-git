DROP TABLE IF EXISTS runner_registration_tokens;
DROP INDEX IF EXISTS idx_runners_org;
DROP TABLE IF EXISTS runners;
DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS job_logs_meta;
DROP INDEX IF EXISTS idx_job_log_lines_lookup;
DROP TABLE IF EXISTS job_log_lines;
DROP INDEX IF EXISTS idx_workflow_jobs_org_run;
DROP TABLE IF EXISTS workflow_jobs;
DROP INDEX IF EXISTS idx_workflow_runs_repo_sha;
DROP INDEX IF EXISTS idx_workflow_runs_repo_created;
DROP INDEX IF EXISTS idx_workflow_runs_org;
DROP TABLE IF EXISTS workflow_runs;

-- Restore the legacy tables from migrations 001/013/014/022 so older code
-- paths keep a table to query.
CREATE TABLE workflow_runs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    workflow TEXT NOT NULL,
    status TEXT NOT NULL,
    conclusion TEXT,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);
CREATE INDEX idx_workflow_runs_organization_id ON workflow_runs(organization_id);

CREATE TABLE workflow_jobs(
    id TEXT PRIMARY KEY,
    workflow_run_id TEXT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','in_progress','completed','failed')),
    conclusion TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);
CREATE INDEX idx_workflow_jobs_org_run ON workflow_jobs(organization_id, workflow_run_id);

CREATE TABLE job_log_lines(
    id INTEGER PRIMARY KEY,
    organization_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    job_id TEXT NOT NULL REFERENCES workflow_jobs(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL DEFAULT 0,
    line_number INTEGER NOT NULL,
    stream TEXT NOT NULL DEFAULT 'stdout' CHECK(stream IN ('stdout','stderr')),
    text TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(job_id, line_number)
);
CREATE INDEX idx_job_log_lines_lookup ON job_log_lines(organization_id, job_id, line_number);

CREATE TABLE job_logs_meta(
    job_id TEXT PRIMARY KEY REFERENCES workflow_jobs(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL,
    total_lines INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'running' CHECK(status IN ('running','success','failure','cancelled')),
    archived_at TIMESTAMP
);

CREATE TABLE artifacts (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
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
