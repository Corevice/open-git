DROP TABLE IF EXISTS pull_request_review_comments;

PRAGMA foreign_keys=OFF;

ALTER TABLE reviews RENAME TO reviews_old;

CREATE TABLE reviews (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    pull_request_id TEXT NOT NULL REFERENCES pull_requests(id),
    reviewer_id TEXT NOT NULL REFERENCES users(id),
    state TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    submitted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reviews_organization_id ON reviews(organization_id);

INSERT INTO reviews (id, organization_id, pull_request_id, reviewer_id, state, body, submitted_at)
SELECT id, organization_id, pull_request_id, reviewer_id, state, body, submitted_at
FROM reviews_old;

DROP TABLE reviews_old;

ALTER TABLE pull_requests RENAME TO pull_requests_old;

CREATE TABLE pull_requests (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    number INTEGER NOT NULL,
    head_ref TEXT NOT NULL,
    base_ref TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'open',
    merged_at TIMESTAMP,
    author_id TEXT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(repository_id, number)
);

CREATE INDEX idx_pull_requests_organization_id ON pull_requests(organization_id);

INSERT INTO pull_requests (
    id, organization_id, repository_id, number, head_ref, base_ref, state, merged_at, author_id, created_at
)
SELECT id, organization_id, repository_id, number, head_ref, base_ref, state, merged_at, author_id, created_at
FROM pull_requests_old;

DROP TABLE pull_requests_old;

PRAGMA foreign_keys=ON;
