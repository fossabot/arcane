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
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, url, auth_type, username, token, ssh_key, description, enabled, created_at, updated_at, ssh_host_key_verification;

-- name: GetGitRepositoryByID :one
SELECT id, name, url, auth_type, username, token, ssh_key, description, enabled, created_at, updated_at, ssh_host_key_verification
FROM git_repositories
WHERE id = ?
LIMIT 1;

-- name: GetGitRepositoryByName :one
SELECT id, name, url, auth_type, username, token, ssh_key, description, enabled, created_at, updated_at, ssh_host_key_verification
FROM git_repositories
WHERE name = ?
LIMIT 1;

-- name: ListGitRepositories :many
SELECT id, name, url, auth_type, username, token, ssh_key, description, enabled, created_at, updated_at, ssh_host_key_verification
FROM git_repositories;

-- name: SaveGitRepository :one
UPDATE git_repositories
SET
    name = ?,
    url = ?,
    auth_type = ?,
    username = ?,
    token = ?,
    ssh_key = ?,
    description = ?,
    enabled = ?,
    ssh_host_key_verification = ?,
    updated_at = ?
WHERE id = ?
RETURNING id, name, url, auth_type, username, token, ssh_key, description, enabled, created_at, updated_at, ssh_host_key_verification;

-- name: DeleteGitRepositoryByID :execrows
DELETE FROM git_repositories
WHERE id = ?;

-- name: CountGitOpsSyncsByRepositoryID :one
SELECT COUNT(*)
FROM gitops_syncs
WHERE repository_id = ?;
