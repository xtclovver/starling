ALTER TABLE users ADD COLUMN is_admin  BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN is_banned BOOLEAN NOT NULL DEFAULT FALSE;

-- Retroactively promote the first registered user to admin
UPDATE users SET is_admin = TRUE
WHERE id = (SELECT id FROM users WHERE deleted_at IS NULL ORDER BY created_at ASC LIMIT 1);
