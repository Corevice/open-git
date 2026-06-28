ALTER TABLE pull_requests ADD COLUMN title TEXT NOT NULL DEFAULT '';
ALTER TABLE pull_requests ADD COLUMN body TEXT NOT NULL DEFAULT '';
ALTER TABLE pull_requests ADD COLUMN draft INTEGER NOT NULL DEFAULT 0;
ALTER TABLE pull_requests ADD COLUMN merged_by TEXT REFERENCES users(id);
ALTER TABLE pull_requests ADD COLUMN head_sha TEXT NOT NULL DEFAULT '';
ALTER TABLE pull_requests ADD COLUMN base_sha TEXT NOT NULL DEFAULT '';
ALTER TABLE pull_requests ADD COLUMN merge_commit_sha TEXT;
ALTER TABLE pull_requests ADD COLUMN mergeable INTEGER;
ALTER TABLE pull_requests ADD COLUMN mergeable_state TEXT NOT NULL DEFAULT 'unknown';
ALTER TABLE pull_requests ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE TABLE IF NOT EXISTS pull_request_review_comments (
    id TEXT PRIMARY KEY,
    review_id TEXT REFERENCES reviews(id),
    pull_request_id TEXT NOT NULL REFERENCES pull_requests(id),
    author_id TEXT NOT NULL REFERENCES users(id),
    path TEXT NOT NULL DEFAULT '',
    diff_hunk TEXT NOT NULL DEFAULT '',
    line INTEGER NOT NULL DEFAULT 0,
    side TEXT NOT NULL DEFAULT 'RIGHT',
    body TEXT NOT NULL DEFAULT '',
    in_reply_to_id TEXT REFERENCES pull_request_review_comments(id),
    resolved INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE reviews ADD COLUMN commit_sha TEXT NOT NULL DEFAULT '';
