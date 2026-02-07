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
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING id, name, description, key_hash, key_prefix, user_id, environment_id, expires_at, last_used_at, created_at, updated_at;

-- name: GetApiKeyByID :one
SELECT id, name, description, key_hash, key_prefix, user_id, environment_id, expires_at, last_used_at, created_at, updated_at
FROM api_keys
WHERE id = ?
LIMIT 1;

-- name: ListApiKeys :many
SELECT id, name, description, key_hash, key_prefix, user_id, environment_id, expires_at, last_used_at, created_at, updated_at
FROM api_keys;

-- name: ListApiKeysByPrefix :many
SELECT id, name, description, key_hash, key_prefix, user_id, environment_id, expires_at, last_used_at, created_at, updated_at
FROM api_keys
WHERE key_prefix = ?;

-- name: UpdateApiKey :exec
UPDATE api_keys
SET
    name = ?,
    description = ?,
    expires_at = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteApiKeyByID :execrows
DELETE FROM api_keys
WHERE id = ?;

-- name: TouchApiKeyLastUsed :exec
UPDATE api_keys
SET
    last_used_at = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;
