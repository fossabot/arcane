-- name: GetAppriseSettings :one
SELECT id, api_url, enabled, image_update_tag, container_update_tag, created_at, updated_at
FROM apprise_settings
ORDER BY id ASC
LIMIT 1;

-- name: CreateAppriseSettings :one
INSERT INTO apprise_settings (api_url, enabled, image_update_tag, container_update_tag)
VALUES ($1, $2, $3, $4)
RETURNING id, api_url, enabled, image_update_tag, container_update_tag, created_at, updated_at;

-- name: UpdateAppriseSettings :one
UPDATE apprise_settings
SET api_url = $2,
    enabled = $3,
    image_update_tag = $4,
    container_update_tag = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, api_url, enabled, image_update_tag, container_update_tag, created_at, updated_at;
