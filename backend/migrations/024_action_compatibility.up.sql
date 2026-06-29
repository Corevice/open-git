CREATE TABLE IF NOT EXISTS action_verifications (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    "trigger" varchar NOT NULL,
    status varchar NOT NULL,
    requested_by TEXT REFERENCES users(id),
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_action_verifications_organization_id ON action_verifications(organization_id);

CREATE TABLE IF NOT EXISTS action_compatibility_results (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    repository_id TEXT REFERENCES repositories(id),
    action_name varchar NOT NULL,
    action_version varchar NOT NULL,
    status varchar NOT NULL DEFAULT 'untested',
    note text,
    golden_diff jsonb,
    verified_at timestamptz,
    verification_id TEXT REFERENCES action_verifications(id),
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, action_name, action_version)
);

CREATE INDEX IF NOT EXISTS idx_action_compatibility_results_organization_id ON action_compatibility_results(organization_id);

CREATE TABLE IF NOT EXISTS action_cache_entries (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    action_name varchar NOT NULL,
    resolved_ref varchar NOT NULL,
    storage_path varchar NOT NULL,
    cached_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, action_name, resolved_ref)
);

CREATE INDEX IF NOT EXISTS idx_action_cache_entries_organization_id ON action_cache_entries(organization_id);
