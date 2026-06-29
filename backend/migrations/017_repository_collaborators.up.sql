CREATE TABLE repository_collaborators (
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    permission TEXT NOT NULL CHECK (permission IN ('read', 'write', 'admin')),
    PRIMARY KEY (repository_id, user_id)
);

CREATE INDEX idx_repository_collaborators_repository_id ON repository_collaborators(repository_id);
