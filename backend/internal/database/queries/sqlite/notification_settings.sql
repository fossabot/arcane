-- name: ListNotificationSettings :many
SELECT id, provider, enabled, config, created_at, updated_at
FROM notification_settings
ORDER BY id ASC;

-- name: GetNotificationSettingByProvider :one
SELECT id, provider, enabled, config, created_at, updated_at
FROM notification_settings
WHERE provider = ?
LIMIT 1;

-- name: CreateNotificationSetting :one
INSERT INTO notification_settings (provider, enabled, config)
VALUES (?, ?, ?)
RETURNING id, provider, enabled, config, created_at, updated_at;

-- name: UpdateNotificationSetting :one
UPDATE notification_settings
SET enabled = ?,
    config = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, provider, enabled, config, created_at, updated_at;

-- name: DeleteNotificationSettingByProvider :execrows
DELETE FROM notification_settings
WHERE provider = ?;
