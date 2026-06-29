ALTER TABLE audit_logs ADD COLUMN ip_address TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN user_agent TEXT NOT NULL DEFAULT '';

CREATE TABLE security_advisories (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT REFERENCES repositories(id),
    ghsa_id TEXT NOT NULL,
    cve_id TEXT,
    severity TEXT NOT NULL CHECK(severity IN ('critical', 'high', 'medium', 'low')),
    summary TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    affected_package TEXT NOT NULL DEFAULT '',
    affected_versions TEXT NOT NULL DEFAULT '',
    patched_versions TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL DEFAULT 'open' CHECK(state IN ('open', 'acknowledged', 'resolved', 'dismissed')),
    dismissed_reason TEXT CHECK(dismissed_reason IS NULL OR dismissed_reason IN ('no_bandwidth', 'tolerable_risk', 'inaccurate', 'not_used')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, ghsa_id)
);

CREATE INDEX idx_security_advisories_org_state ON security_advisories(organization_id, state);

CREATE TABLE dependabot_alerts (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    alert_number INTEGER NOT NULL,
    advisory_id TEXT NOT NULL REFERENCES security_advisories(id),
    manifest_path TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL DEFAULT 'open' CHECK(state IN ('open', 'dismissed', 'fixed')),
    auto_dismissed_at TIMESTAMP,
    UNIQUE(repository_id, alert_number)
);

CREATE INDEX idx_dependabot_alerts_org_repo ON dependabot_alerts(organization_id, repository_id);

CREATE TABLE secret_scanning_alerts (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    secret_type TEXT NOT NULL DEFAULT '',
    commit_sha TEXT NOT NULL DEFAULT '',
    file_path TEXT NOT NULL DEFAULT '',
    line INTEGER NOT NULL DEFAULT 0,
    state TEXT NOT NULL DEFAULT 'open' CHECK(state IN ('open', 'resolved', 'false_positive')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_secret_scanning_alerts_org_repo ON secret_scanning_alerts(organization_id, repository_id);

CREATE TABLE scan_jobs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    type TEXT NOT NULL CHECK(type IN ('dependency', 'secret')),
    status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued', 'running', 'completed', 'scan_failed', 'parse_error')),
    retry_count INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    error TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_scan_jobs_org_repo ON scan_jobs(organization_id, repository_id);
