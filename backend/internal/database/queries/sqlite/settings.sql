-- name: ListSettings :many
SELECT key, value
FROM settings;

-- name: GetSetting :one
SELECT key, value
FROM settings
WHERE key = ?
LIMIT 1;

-- name: UpsertSetting :exec
INSERT INTO settings (key, value)
VALUES (?, ?)
ON CONFLICT (key) DO UPDATE SET value = excluded.value;

-- name: InsertSettingIfNotExists :exec
INSERT INTO settings (key, value)
VALUES (?, ?)
ON CONFLICT (key) DO NOTHING;

-- name: DeleteSetting :execrows
DELETE FROM settings
WHERE key = ?;

-- name: UpdateSettingKey :exec
UPDATE settings
SET key = ?
WHERE key = ?;

-- name: DeleteSettingsNotIn :execrows
DELETE FROM settings
WHERE key NOT IN (sqlc.slice('keys'));
