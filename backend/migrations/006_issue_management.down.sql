DROP INDEX IF EXISTS idx_issue_assignees_issue_id;
DROP TABLE IF EXISTS issue_assignees;

DROP INDEX IF EXISTS idx_issue_labels_issue_id;
DROP TABLE IF EXISTS issue_labels;

-- SQLite does not support dropping columns from existing tables without table
-- recreation. Column additions from this migration are therefore not reversible
-- on SQLite. On PostgreSQL, each column can be dropped individually:

ALTER TABLE milestones DROP COLUMN updated_at;
ALTER TABLE milestones DROP COLUMN closed_at;
ALTER TABLE milestones DROP COLUMN closed_issues;
ALTER TABLE milestones DROP COLUMN open_issues;
ALTER TABLE milestones DROP COLUMN number;

ALTER TABLE labels DROP COLUMN description;

ALTER TABLE comments DROP COLUMN updated_at;

ALTER TABLE issues DROP COLUMN comments_count;
ALTER TABLE issues DROP COLUMN milestone_id;
ALTER TABLE issues DROP COLUMN state_reason;
ALTER TABLE issues DROP COLUMN closed_at;
ALTER TABLE issues DROP COLUMN updated_at;
