-- name: CreateGitRepository :one
INSERT INTO git_repositories (
    id,
    name,
    url,
    auth_type,
    username,
    token,
    ssh_key,
    description,
    enabled,
    ssh_host_key_verification,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, name, url, auth_type, username, token, ssh_key, description, enabled, created_at, updated_at, ssh_host_key_verification;

-- name: GetGitRepositoryByID :one
SELECT id, name, url, auth_type, username, token, ssh_key, description, enabled, created_at, updated_at, ssh_host_key_verification
FROM git_repositories
WHERE id = $1
LIMIT 1;

-- name: GetGitRepositoryByName :one
SELECT id, name, url, auth_type, username, token, ssh_key, description, enabled, created_at, updated_at, ssh_host_key_verification
FROM git_repositories
WHERE name = $1
LIMIT 1;

-- name: ListGitRepositories :many
SELECT id, name, url, auth_type, username, token, ssh_key, description, enabled, created_at, updated_at, ssh_host_key_verification
FROM git_repositories;

-- name: SaveGitRepository :one
UPDATE git_repositories
SET
    name = $1,
    url = $2,
    auth_type = $3,
    username = $4,
    token = $5,
    ssh_key = $6,
    description = $7,
    enabled = $8,
    ssh_host_key_verification = $9,
    updated_at = $10
WHERE id = $11
RETURNING id, name, url, auth_type, username, token, ssh_key, description, enabled, created_at, updated_at, ssh_host_key_verification;

-- name: DeleteGitRepositoryByID :execrows
DELETE FROM git_repositories
WHERE id = $1;

-- name: CountGitOpsSyncsByRepositoryID :one
SELECT COUNT(*)::bigint
FROM gitops_syncs
WHERE repository_id = $1;
