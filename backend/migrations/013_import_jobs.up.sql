CREATE TABLE import_jobs (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    created_by UUID NOT NULL REFERENCES users(id),
    source_url TEXT NOT NULL,
    target_repository_id UUID REFERENCES repositories(id),
    target_name VARCHAR(100) NOT NULL,
    include JSONB NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','running','paused','completed','failed','cancelled')),
    phase TEXT NOT NULL DEFAULT 'clone' CHECK(phase IN ('clone','metadata','issues','pull_requests','wiki','done')),
    progress JSONB NOT NULL DEFAULT '{}',
    token_secret_ref TEXT,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_import_jobs_org_status ON import_jobs(organization_id, status);
CREATE INDEX idx_import_jobs_org_created ON import_jobs(organization_id, created_at);

CREATE TABLE import_user_mappings (
    id UUID PRIMARY KEY,
    import_job_id UUID NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    github_login VARCHAR NOT NULL,
    github_display_name VARCHAR NOT NULL,
    local_user_id UUID REFERENCES users(id),
    UNIQUE(import_job_id, github_login)
);

CREATE TABLE import_phase_checkpoints (
    import_job_id UUID NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    phase TEXT NOT NULL,
    last_cursor TEXT NOT NULL DEFAULT '',
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY(import_job_id, phase)
);
