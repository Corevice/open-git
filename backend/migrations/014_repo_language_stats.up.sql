CREATE TABLE IF NOT EXISTS repo_language_stats (
  repo_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  language VARCHAR(100) NOT NULL,
  bytes BIGINT NOT NULL DEFAULT 0,
  percentage NUMERIC(5,2) NOT NULL DEFAULT 0,
  computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (repo_id, language)
);
