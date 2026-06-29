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
