CREATE TABLE mcp_verification_runs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    repository_id TEXT,
    triggered_by TEXT,
    status TEXT NOT NULL DEFAULT 'queued',
    overall_status TEXT,
    targets TEXT NOT NULL DEFAULT '[]',
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE mcp_verification_checks (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES mcp_verification_runs(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL,
    check_id TEXT NOT NULL,
    category TEXT NOT NULL,
    status TEXT NOT NULL,
    expected TEXT,
    actual TEXT,
    error TEXT,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mcp_runs_org_created ON mcp_verification_runs (organization_id, created_at DESC);
CREATE INDEX idx_mcp_checks_run ON mcp_verification_checks (run_id);
CREATE UNIQUE INDEX uidx_mcp_runs_org_active ON mcp_verification_runs (organization_id) WHERE status IN ('queued', 'running');
