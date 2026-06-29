CREATE TABLE compat_test_run (
    id uuid PRIMARY KEY,
    suite varchar NOT NULL,
    status varchar NOT NULL DEFAULT 'queued',
    triggered_by uuid REFERENCES users(id),
    organization_id uuid NOT NULL,
    total_endpoints int NOT NULL DEFAULT 0,
    passing int NOT NULL DEFAULT 0,
    failing int NOT NULL DEFAULT 0,
    unimplemented int NOT NULL DEFAULT 0,
    coverage_rate numeric(5,4) NOT NULL DEFAULT 0,
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE compat_endpoint_result (
    id uuid PRIMARY KEY,
    run_id uuid NOT NULL REFERENCES compat_test_run(id) ON DELETE CASCADE,
    method varchar NOT NULL,
    path varchar NOT NULL,
    status varchar NOT NULL,
    checks jsonb,
    diff jsonb
);

CREATE TABLE compat_golden_fixture (
    id uuid PRIMARY KEY,
    method varchar NOT NULL,
    path varchar NOT NULL,
    response_schema jsonb,
    sample_response jsonb,
    github_api_version varchar NOT NULL DEFAULT '2022-11-28',
    UNIQUE(method, path)
);
