-- Personal organizations. Personal repositories use the owner's user id as
-- their organization id, and ~20 tables carry a foreign key to
-- organizations(id). SQLite does not enforce foreign keys (the app does not
-- enable the pragma), so this was invisible there, but on PostgreSQL every
-- org-scoped insert for a personal repo (repositories, issues, audit_logs,
-- webhooks, ...) fails the FK because no organizations row exists for the user.
--
-- Give every existing user a personal organization whose id equals the user id
-- (new users get theirs at registration). Skipped where a row already exists
-- by id, or where the login is already taken by a real organization.
INSERT INTO organizations (id, login, name, plan_tier, created_at)
SELECT u.id, u.login, u.login, 'free', u.created_at
FROM users u
WHERE NOT EXISTS (SELECT 1 FROM organizations o WHERE o.id = u.id)
  AND NOT EXISTS (SELECT 1 FROM organizations o2 WHERE o2.login = u.login);
