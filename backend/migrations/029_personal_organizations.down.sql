-- Remove personal organizations (those whose id matches a user id). Real
-- organizations (id not equal to any user id) are left intact.
DELETE FROM organizations
WHERE id IN (SELECT id FROM users);
