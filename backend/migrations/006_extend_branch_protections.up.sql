ALTER TABLE branch_protections ADD COLUMN dismiss_stale_reviews INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN require_code_owner_reviews INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN required_status_checks_strict INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN enforce_admins INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN allow_force_pushes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN allow_deletions INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN required_linear_history INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN required_conversation_resolution INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN updated_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_bp_org_repo ON branch_protections (organization_id, repository_id);

CREATE TABLE branch_protection_required_checks (
    id TEXT PRIMARY KEY,
    rule_id TEXT NOT NULL REFERENCES branch_protections(id) ON DELETE CASCADE,
    context TEXT NOT NULL,
    UNIQUE(rule_id, context)
);
