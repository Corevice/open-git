CREATE TABLE compat_test_run (
    id TEXT PRIMARY KEY,
    suite TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    triggered_by TEXT REFERENCES users(id),
    organization_id TEXT NOT NULL,
    total_endpoints INTEGER NOT NULL DEFAULT 0,
    passing INTEGER NOT NULL DEFAULT 0,
    failing INTEGER NOT NULL DEFAULT 0,
    unimplemented INTEGER NOT NULL DEFAULT 0,
    coverage_rate REAL NOT NULL DEFAULT 0,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE compat_endpoint_result (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES compat_test_run(id) ON DELETE CASCADE,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    status TEXT NOT NULL,
    checks TEXT,
    diff TEXT
);

CREATE TABLE compat_golden_fixture (
    id TEXT PRIMARY KEY,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    response_schema TEXT,
    sample_response TEXT,
    github_api_version TEXT NOT NULL DEFAULT '2022-11-28',
    UNIQUE(method, path)
);
