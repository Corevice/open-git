-- Migration numbering: prior migrations in this repo (verified on disk):
--   001_initial_schema through 017_repository_collaborators
-- This is the next sequential migration (018). Task docs may reference "002" for
-- this feature slice, but 002 is occupied by webhook_delivery_status; golang-migrate
-- requires monotonic filenames — do not reuse 002.
--
-- NOTE: SQLite does not support ADD COLUMN IF NOT EXISTS. Re-running this migration
-- after partial failure will error on duplicate columns; down migration cannot remove
-- those columns (see down.sql).

ALTER TABLE branch_protections ADD COLUMN dismiss_stale_reviews INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN require_code_owner_reviews INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN required_status_checks_strict INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN enforce_admins INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN allow_force_pushes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN allow_deletions INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN required_linear_history INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN required_conversation_resolution INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN updated_at TIMESTAMP;

-- organization_id is already indexed by 001_initial_schema (idx_branch_protections_organization_id).
CREATE INDEX IF NOT EXISTS idx_bp_org_repo ON branch_protections (organization_id, repository_id);

CREATE TABLE branch_protection_required_checks (
    id TEXT PRIMARY KEY,
    rule_id TEXT NOT NULL REFERENCES branch_protections(id) ON DELETE CASCADE,
    context TEXT NOT NULL,
    UNIQUE(rule_id, context)
);
