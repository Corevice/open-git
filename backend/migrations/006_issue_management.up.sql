ALTER TABLE issues ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE issues ADD COLUMN closed_at TIMESTAMP;
ALTER TABLE issues ADD COLUMN state_reason TEXT;
-- milestone_id FK is enforced at the application layer; inline REFERENCES would
-- prevent SQLite from dropping the column during migrate down (see migrate_test).
ALTER TABLE issues ADD COLUMN milestone_id TEXT;
ALTER TABLE issues ADD COLUMN comments_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE comments ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE labels ADD COLUMN description TEXT NOT NULL DEFAULT '';

ALTER TABLE milestones ADD COLUMN number INTEGER NOT NULL DEFAULT 0;
ALTER TABLE milestones ADD COLUMN open_issues INTEGER NOT NULL DEFAULT 0;
ALTER TABLE milestones ADD COLUMN closed_issues INTEGER NOT NULL DEFAULT 0;
ALTER TABLE milestones ADD COLUMN closed_at TIMESTAMP;
ALTER TABLE milestones ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- Milestone deduplication (e.g. UNIQUE(repository_id, number)) is enforced at the
-- application layer. SQLite cannot add UNIQUE constraints to existing tables without
-- table recreation, so we avoid that here.

CREATE TABLE issue_labels (
    issue_id TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    label_id TEXT NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, label_id)
);

CREATE INDEX idx_issue_labels_issue_id ON issue_labels(issue_id);

CREATE TABLE issue_assignees (
    issue_id TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, user_id)
);

CREATE INDEX idx_issue_assignees_issue_id ON issue_assignees(issue_id);
