CREATE TABLE IF NOT EXISTS user_preferences (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  theme VARCHAR(10) NOT NULL DEFAULT 'system' CHECK (theme IN ('light', 'dark', 'system')),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
