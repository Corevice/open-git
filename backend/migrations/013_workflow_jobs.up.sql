CREATE TABLE workflow_jobs(
    id TEXT PRIMARY KEY,
    workflow_run_id TEXT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','in_progress','completed','failed')),
    conclusion TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX idx_workflow_jobs_org_run ON workflow_jobs(organization_id, workflow_run_id);
