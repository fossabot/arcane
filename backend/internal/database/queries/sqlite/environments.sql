-- name: CreateEnvironment :one
INSERT INTO environments (
    id,
    name,
    api_url,
    status,
    enabled,
    is_edge,
    last_seen,
    access_token,
    api_key_id,
    created_at,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, api_url, status, enabled, last_seen, created_at, updated_at, access_token, name, api_key_id, is_edge;

-- name: GetEnvironmentByID :one
SELECT id, api_url, status, enabled, last_seen, created_at, updated_at, access_token, name, api_key_id, is_edge
FROM environments
WHERE id = ?
LIMIT 1;

-- name: ListEnvironments :many
SELECT id, api_url, status, enabled, last_seen, created_at, updated_at, access_token, name, api_key_id, is_edge
FROM environments;

-- name: ListRemoteEnvironments :many
SELECT id, api_url, status, enabled, last_seen, created_at, updated_at, access_token, name, api_key_id, is_edge
FROM environments
WHERE id != '0' AND enabled = true;

-- name: PatchEnvironment :one
UPDATE environments
SET
    name = COALESCE(sqlc.narg('name'), name),
    api_url = COALESCE(sqlc.narg('api_url'), api_url),
    status = COALESCE(sqlc.narg('status'), status),
    enabled = COALESCE(sqlc.narg('enabled'), enabled),
    is_edge = COALESCE(sqlc.narg('is_edge'), is_edge),
    last_seen = CASE WHEN sqlc.arg('clear_last_seen') = 1 THEN NULL ELSE COALESCE(sqlc.narg('last_seen'), last_seen) END,
    access_token = CASE WHEN sqlc.arg('clear_access_token') = 1 THEN NULL ELSE COALESCE(sqlc.narg('access_token'), access_token) END,
    api_key_id = CASE WHEN sqlc.arg('clear_api_key_id') = 1 THEN NULL ELSE COALESCE(sqlc.narg('api_key_id'), api_key_id) END,
    updated_at = COALESCE(sqlc.narg('updated_at'), updated_at)
WHERE id = sqlc.arg('id')
RETURNING id, api_url, status, enabled, last_seen, created_at, updated_at, access_token, name, api_key_id, is_edge;

-- name: DeleteEnvironmentByID :execrows
DELETE FROM environments
WHERE id = ?;

-- name: TouchEnvironmentHeartbeatIfStale :execrows
UPDATE environments
SET last_seen = ?,
    status = ?,
    updated_at = ?
WHERE id = ?
  AND (last_seen IS NULL OR last_seen < ?);

-- name: FindEnvironmentIDByApiKeyHash :one
SELECT environments.id
FROM environments
INNER JOIN api_keys ON api_keys.id = environments.api_key_id
WHERE api_keys.key_hash = ?
LIMIT 1;
