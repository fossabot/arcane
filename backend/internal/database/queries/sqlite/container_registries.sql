-- name: CreateContainerRegistry :one
INSERT INTO container_registries (
    id,
    url,
    username,
    token,
    description,
    insecure,
    enabled,
    created_at,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING id, url, username, token, description, insecure, enabled, created_at, updated_at;

-- name: GetContainerRegistryByID :one
SELECT id, url, username, token, description, insecure, enabled, created_at, updated_at
FROM container_registries
WHERE id = ?
LIMIT 1;

-- name: ListContainerRegistries :many
SELECT id, url, username, token, description, insecure, enabled, created_at, updated_at
FROM container_registries;

-- name: ListEnabledContainerRegistries :many
SELECT id, url, username, token, description, insecure, enabled, created_at, updated_at
FROM container_registries
WHERE enabled = true;

-- name: UpdateContainerRegistry :exec
UPDATE container_registries
SET
    url = ?,
    username = ?,
    token = ?,
    description = ?,
    insecure = ?,
    enabled = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteContainerRegistryByID :execrows
DELETE FROM container_registries
WHERE id = ?;
