-- User preferences (theme, etc.) were implemented in the repository, usecases
-- and handler but no migration ever created the table, so GET/PUT
-- /user/preferences failed with "no such table: user_preferences" (500).
--
-- user_id is the int64 user identifier used throughout the auth layer (the JWT
-- subject), not the TEXT uuid stored in users.id, so this column is BIGINT and
-- has no foreign key to users.
CREATE TABLE user_preferences (
    user_id BIGINT PRIMARY KEY,
    theme TEXT NOT NULL DEFAULT 'system',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
