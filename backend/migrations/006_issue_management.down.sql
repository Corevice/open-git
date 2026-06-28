DROP INDEX IF EXISTS idx_issue_assignees_issue_id;
DROP TABLE IF EXISTS issue_assignees;

DROP INDEX IF EXISTS idx_issue_labels_issue_id;
DROP TABLE IF EXISTS issue_labels;

-- SQLite 3.35+ supports DROP COLUMN; project tests use go-sqlite3 with DROP COLUMN
-- (see 002_webhook_delivery_status.down.sql). milestone_id has no inline REFERENCES
-- so the column can be dropped here as well.

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
