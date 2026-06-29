CREATE TABLE mcp_verification_runs (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    repository_id uuid NULL,
    triggered_by uuid NULL,
    status text NOT NULL DEFAULT 'queued',
    overall_status text NULL,
    targets jsonb NOT NULL DEFAULT '[]',
    started_at timestamptz NULL,
    finished_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE mcp_verification_checks (
    id uuid PRIMARY KEY,
    run_id uuid NOT NULL REFERENCES mcp_verification_runs(id) ON DELETE CASCADE,
    organization_id uuid NOT NULL,
    check_id text NOT NULL,
    category text NOT NULL,
    status text NOT NULL,
    expected jsonb NULL,
    actual jsonb NULL,
    error text NULL,
    duration_ms int NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mcp_runs_org_created ON mcp_verification_runs (organization_id, created_at DESC);
CREATE INDEX idx_mcp_checks_run ON mcp_verification_checks (run_id);
CREATE UNIQUE INDEX uidx_mcp_runs_org_active ON mcp_verification_runs (organization_id) WHERE status IN ('queued', 'running');
