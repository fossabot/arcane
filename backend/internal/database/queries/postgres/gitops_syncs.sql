-- name: CreateGitOpsSync :one
INSERT INTO gitops_syncs (
    id,
    name,
    environment_id,
    repository_id,
    branch,
    compose_path,
    project_name,
    project_id,
    auto_sync,
    sync_interval,
    last_sync_at,
    last_sync_status,
    last_sync_error,
    last_sync_commit,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING id, name, repository_id, branch, compose_path, project_name, project_id, auto_sync, sync_interval, last_sync_at, last_sync_status, last_sync_error, last_sync_commit, environment_id, created_at, updated_at;

-- name: GetGitOpsSyncByID :one
SELECT id, name, repository_id, branch, compose_path, project_name, project_id, auto_sync, sync_interval, last_sync_at, last_sync_status, last_sync_error, last_sync_commit, environment_id, created_at, updated_at
FROM gitops_syncs
WHERE id = $1
LIMIT 1;

-- name: ListGitOpsSyncs :many
SELECT id, name, repository_id, branch, compose_path, project_name, project_id, auto_sync, sync_interval, last_sync_at, last_sync_status, last_sync_error, last_sync_commit, environment_id, created_at, updated_at
FROM gitops_syncs;

-- name: ListGitOpsSyncsByEnvironment :many
SELECT id, name, repository_id, branch, compose_path, project_name, project_id, auto_sync, sync_interval, last_sync_at, last_sync_status, last_sync_error, last_sync_commit, environment_id, created_at, updated_at
FROM gitops_syncs
WHERE environment_id = $1;

-- name: ListAutoSyncGitOpsSyncs :many
SELECT id, name, repository_id, branch, compose_path, project_name, project_id, auto_sync, sync_interval, last_sync_at, last_sync_status, last_sync_error, last_sync_commit, environment_id, created_at, updated_at
FROM gitops_syncs
WHERE auto_sync = true;

-- name: SaveGitOpsSync :one
UPDATE gitops_syncs
SET
    name = $1,
    environment_id = $2,
    repository_id = $3,
    branch = $4,
    compose_path = $5,
    project_name = $6,
    project_id = $7,
    auto_sync = $8,
    sync_interval = $9,
    last_sync_at = $10,
    last_sync_status = $11,
    last_sync_error = $12,
    last_sync_commit = $13,
    updated_at = $14
WHERE id = $15
RETURNING id, name, repository_id, branch, compose_path, project_name, project_id, auto_sync, sync_interval, last_sync_at, last_sync_status, last_sync_error, last_sync_commit, environment_id, created_at, updated_at;

-- name: DeleteGitOpsSyncByID :execrows
DELETE FROM gitops_syncs
WHERE id = $1;

-- name: UpdateGitOpsSyncInterval :exec
UPDATE gitops_syncs
SET sync_interval = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2;

-- name: UpdateGitOpsSyncProjectID :exec
UPDATE gitops_syncs
SET project_id = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2;

-- name: UpdateGitOpsSyncStatus :exec
UPDATE gitops_syncs
SET
    last_sync_at = $1,
    last_sync_status = $2,
    last_sync_error = $3,
    last_sync_commit = $4,
    updated_at = $1
WHERE id = $5;

-- name: SetProjectGitOpsManagedBy :exec
UPDATE projects
SET gitops_managed_by = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2;

-- name: ClearProjectGitOpsManagedByIfMatches :exec
UPDATE projects
SET gitops_managed_by = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND gitops_managed_by = $2;
