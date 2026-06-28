ALTER TABLE workflow_runs ADD COLUMN run_number INTEGER NOT NULL DEFAULT 0;
ALTER TABLE workflow_runs ADD COLUMN name TEXT NOT NULL DEFAULT '';
ALTER TABLE workflow_runs ADD COLUMN event TEXT NOT NULL DEFAULT '';
ALTER TABLE workflow_runs ADD COLUMN head_branch TEXT NOT NULL DEFAULT '';
ALTER TABLE workflow_runs ADD COLUMN head_sha TEXT NOT NULL DEFAULT '';
ALTER TABLE workflow_runs ADD COLUMN actor_id TEXT REFERENCES users(id);

CREATE INDEX IF NOT EXISTS idx_workflow_runs_repo_created ON workflow_runs(repository_id, created_at DESC);

CREATE TABLE workflow_jobs (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES workflow_runs(id),
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    conclusion TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    runner_name TEXT NOT NULL DEFAULT '',
    runner_labels TEXT NOT NULL DEFAULT '[]'
);

CREATE INDEX idx_workflow_jobs_run_id ON workflow_jobs(run_id);

CREATE TABLE workflow_steps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT NOT NULL REFERENCES workflow_jobs(id),
    number INTEGER NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    conclusion TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX idx_workflow_steps_job_id ON workflow_steps(job_id);

CREATE TABLE artifacts (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES workflow_runs(id),
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    name TEXT NOT NULL,
    size_in_bytes INTEGER NOT NULL DEFAULT 0,
    storage_key TEXT NOT NULL DEFAULT '',
    expired INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);

CREATE INDEX idx_artifacts_run_id ON artifacts(run_id);
