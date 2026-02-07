-- name: GetAppriseSettings :one
SELECT id, api_url, enabled, image_update_tag, container_update_tag, created_at, updated_at
FROM apprise_settings
ORDER BY id ASC
LIMIT 1;

-- name: CreateAppriseSettings :one
INSERT INTO apprise_settings (api_url, enabled, image_update_tag, container_update_tag)
VALUES (?, ?, ?, ?)
RETURNING id, api_url, enabled, image_update_tag, container_update_tag, created_at, updated_at;

-- name: UpdateAppriseSettings :one
UPDATE apprise_settings
SET api_url = ?,
    enabled = ?,
    image_update_tag = ?,
    container_update_tag = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, api_url, enabled, image_update_tag, container_update_tag, created_at, updated_at;
