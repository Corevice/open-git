-- Actions/CI runtime schema. The workflow_runs/workflow_jobs tables from
-- migrations 001/013 never matched what the code expects (missing head_sha,
-- head_branch, event, actor_login, run_number on runs; missing runner
-- assignment/locking/timing columns on jobs; a CHECK that rejects the
-- 'cancelled' status the code sets), and nothing in any production code path
-- has ever inserted into them — they have always been empty. Recreate them to
-- the shape the repositories and entities use.
--
-- job_log_lines / job_logs_meta / artifacts hold FKs into these tables, so
-- they are dropped and recreated (identically to migrations 014/022) around
-- the rebuild; with the referenced tables permanently empty, they can hold no
-- rows either. DROP+CREATE keeps this portable across SQLite/Postgres.
--
-- No FK on organization_id anywhere here: personal repositories use the
-- owner's user id as the organization id and have no organizations row.

DROP TABLE IF EXISTS job_log_lines;
DROP TABLE IF EXISTS job_logs_meta;
DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS workflow_jobs;
DROP TABLE IF EXISTS workflow_runs;

CREATE TABLE workflow_runs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    workflow_id TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    workflow TEXT NOT NULL,
    head_sha TEXT NOT NULL DEFAULT '',
    head_branch TEXT NOT NULL DEFAULT '',
    event TEXT NOT NULL DEFAULT '',
    actor_login TEXT NOT NULL DEFAULT '',
    run_number INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    conclusion TEXT,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_workflow_runs_org ON workflow_runs(organization_id);
CREATE INDEX idx_workflow_runs_repo_created ON workflow_runs(repository_id, created_at);
CREATE INDEX idx_workflow_runs_repo_sha ON workflow_runs(repository_id, head_sha);

CREATE TABLE workflow_jobs (
    id TEXT PRIMARY KEY,
    workflow_run_id TEXT REFERENCES workflow_runs(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'queued',
    conclusion TEXT NOT NULL DEFAULT '',
    assigned_runner_id TEXT,
    runs_on TEXT NOT NULL DEFAULT '[]',
    acquire_lock_version INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    timeout_minutes INTEGER NOT NULL DEFAULT 60,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_workflow_jobs_org_run ON workflow_jobs(organization_id, workflow_run_id);

-- Recreated identically to migration 014.
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

-- Recreated identically to migration 022, minus the organizations FK (personal
-- repos have no organizations row) — same reasoning as the tables above.
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

-- Self-hosted runner registry (runner registration/heartbeat API).
CREATE TABLE runners (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    labels TEXT NOT NULL DEFAULT '[]',
    os TEXT NOT NULL DEFAULT '',
    arch TEXT NOT NULL DEFAULT '',
    runner_type TEXT NOT NULL DEFAULT 'official',
    status TEXT NOT NULL DEFAULT 'offline',
    last_seen_at TIMESTAMP,
    ephemeral BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_runners_org ON runners(organization_id);

CREATE TABLE runner_registration_tokens (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP
);
