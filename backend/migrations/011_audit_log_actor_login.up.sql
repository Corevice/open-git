ALTER TABLE audit_logs ADD COLUMN actor_login TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_audit_logs_org_created ON audit_logs(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_org_action ON audit_logs(organization_id, action);
