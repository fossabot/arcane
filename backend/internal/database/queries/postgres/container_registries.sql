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
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
RETURNING id, url, username, token, description, insecure, enabled, created_at, updated_at;

-- name: GetContainerRegistryByID :one
SELECT id, url, username, token, description, insecure, enabled, created_at, updated_at
FROM container_registries
WHERE id = $1
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
    url = $2,
    username = $3,
    token = $4,
    description = $5,
    insecure = $6,
    enabled = $7,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteContainerRegistryByID :execrows
DELETE FROM container_registries
WHERE id = $1;
