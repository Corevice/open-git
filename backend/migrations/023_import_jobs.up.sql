CREATE TABLE IF NOT EXISTS import_jobs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    created_by TEXT NOT NULL REFERENCES users(id),
    source_url TEXT NOT NULL,
    target_repository_id TEXT REFERENCES repositories(id),
    target_name TEXT NOT NULL,
    include jsonb NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','running','paused','completed','failed','cancelled')),
    phase TEXT NOT NULL DEFAULT 'clone' CHECK(phase IN ('clone','metadata','issues','pull_requests','wiki','done')),
    progress jsonb NOT NULL DEFAULT '{}',
    token_secret_ref TEXT,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_import_jobs_org_status ON import_jobs(organization_id, status);
CREATE INDEX IF NOT EXISTS idx_import_jobs_org_created ON import_jobs(organization_id, created_at);

CREATE TABLE IF NOT EXISTS import_user_mappings (
    id TEXT PRIMARY KEY,
    import_job_id TEXT NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    github_login TEXT NOT NULL,
    github_display_name TEXT NOT NULL DEFAULT '',
    local_user_id TEXT REFERENCES users(id),
    UNIQUE(import_job_id, github_login)
);

CREATE TABLE IF NOT EXISTS import_phase_checkpoints (
    import_job_id TEXT NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    phase TEXT NOT NULL,
    last_cursor TEXT NOT NULL DEFAULT '',
    completed INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY(import_job_id, phase)
);
