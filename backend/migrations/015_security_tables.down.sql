DROP TABLE IF EXISTS scan_jobs;
DROP TABLE IF EXISTS secret_scanning_alerts;
DROP TABLE IF EXISTS dependabot_alerts;
DROP TABLE IF EXISTS security_advisories;

PRAGMA foreign_keys=OFF;

DROP INDEX IF EXISTS idx_audit_logs_organization_id;
DROP INDEX IF EXISTS idx_audit_logs_org_created;
DROP INDEX IF EXISTS idx_audit_logs_org_action;

ALTER TABLE audit_logs RENAME TO audit_logs_old;

CREATE TABLE audit_logs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    actor_id TEXT NOT NULL REFERENCES users(id),
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actor_login TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_audit_logs_organization_id ON audit_logs(organization_id);
CREATE INDEX idx_audit_logs_org_created ON audit_logs(organization_id, created_at DESC);
CREATE INDEX idx_audit_logs_org_action ON audit_logs(organization_id, action);

INSERT INTO audit_logs (
    id, organization_id, actor_id, action, target_type, target_id, metadata, created_at, actor_login
)
SELECT id, organization_id, actor_id, action, target_type, target_id, metadata, created_at, actor_login
FROM audit_logs_old;

DROP TABLE audit_logs_old;

PRAGMA foreign_keys=ON;
