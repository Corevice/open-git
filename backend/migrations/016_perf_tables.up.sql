CREATE TABLE perf_benchmarks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_name TEXT NOT NULL,
    environment TEXT NOT NULL CHECK (environment IN ('docker-compose','k8s','ci')),
    status TEXT NOT NULL DEFAULT 'completed',
    slo_result TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    git_sha TEXT,
    metrics JSONB NOT NULL,
    regression JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE perf_slo_thresholds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_name TEXT NOT NULL UNIQUE,
    p95_ms_max INT,
    p99_ms_max INT,
    error_rate_max NUMERIC(5,4),
    throughput_rps_min INT,
    regression_pct_max NUMERIC(5,2),
    updated_at TIMESTAMPTZ
);

CREATE TABLE perf_baselines (
    scenario_name TEXT PRIMARY KEY,
    benchmark_id UUID REFERENCES perf_benchmarks(id),
    set_at TIMESTAMPTZ
);

CREATE TABLE perf_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status TEXT NOT NULL DEFAULT 'queued',
    triggered_by UUID REFERENCES users(id),
    benchmark_id UUID REFERENCES perf_benchmarks(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_perf_benchmarks_scenario_created ON perf_benchmarks(scenario_name, created_at DESC);
CREATE INDEX idx_perf_benchmarks_environment ON perf_benchmarks(environment);
