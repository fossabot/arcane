-- name: CreateApiKey :one
INSERT INTO api_keys (
    id,
    name,
    description,
    key_hash,
    key_prefix,
    user_id,
    environment_id,
    expires_at,
    last_used_at,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
RETURNING id, name, description, key_hash, key_prefix, user_id, environment_id, expires_at, last_used_at, created_at, updated_at;

-- name: GetApiKeyByID :one
SELECT id, name, description, key_hash, key_prefix, user_id, environment_id, expires_at, last_used_at, created_at, updated_at
FROM api_keys
WHERE id = $1
LIMIT 1;

-- name: ListApiKeys :many
SELECT id, name, description, key_hash, key_prefix, user_id, environment_id, expires_at, last_used_at, created_at, updated_at
FROM api_keys;

-- name: ListApiKeysByPrefix :many
SELECT id, name, description, key_hash, key_prefix, user_id, environment_id, expires_at, last_used_at, created_at, updated_at
FROM api_keys
WHERE key_prefix = $1;

-- name: UpdateApiKey :exec
UPDATE api_keys
SET
    name = $2,
    description = $3,
    expires_at = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteApiKeyByID :execrows
DELETE FROM api_keys
WHERE id = $1;

-- name: TouchApiKeyLastUsed :exec
UPDATE api_keys
SET
    last_used_at = $2,
    updated_at = NOW()
WHERE id = $1;
