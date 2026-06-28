DROP TABLE IF EXISTS branch_protection_required_checks;
DROP INDEX IF EXISTS idx_bp_org_repo;

-- SQLite does not support DROP COLUMN; the nine columns added in the up migration
-- (dismiss_stale_reviews, require_code_owner_reviews, required_status_checks_strict,
-- enforce_admins, allow_force_pushes, allow_deletions, required_linear_history,
-- required_conversation_resolution, updated_at) are intentionally left in place on downgrade.
