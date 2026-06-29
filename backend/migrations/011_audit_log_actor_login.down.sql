DROP INDEX IF EXISTS idx_audit_logs_org_action;
DROP INDEX IF EXISTS idx_audit_logs_org_created;
ALTER TABLE audit_logs DROP COLUMN actor_login;
