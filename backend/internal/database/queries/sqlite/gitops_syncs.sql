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
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, environment_id, repository_id, branch, compose_path, project_name, project_id, auto_sync, sync_interval, last_sync_at, last_sync_status, last_sync_error, last_sync_commit, created_at, updated_at;

-- name: GetGitOpsSyncByID :one
SELECT id, name, environment_id, repository_id, branch, compose_path, project_name, project_id, auto_sync, sync_interval, last_sync_at, last_sync_status, last_sync_error, last_sync_commit, created_at, updated_at
FROM gitops_syncs
WHERE id = ?
LIMIT 1;

-- name: ListGitOpsSyncs :many
SELECT id, name, environment_id, repository_id, branch, compose_path, project_name, project_id, auto_sync, sync_interval, last_sync_at, last_sync_status, last_sync_error, last_sync_commit, created_at, updated_at
FROM gitops_syncs;

-- name: ListGitOpsSyncsByEnvironment :many
SELECT id, name, environment_id, repository_id, branch, compose_path, project_name, project_id, auto_sync, sync_interval, last_sync_at, last_sync_status, last_sync_error, last_sync_commit, created_at, updated_at
FROM gitops_syncs
WHERE environment_id = ?;

-- name: ListAutoSyncGitOpsSyncs :many
SELECT id, name, environment_id, repository_id, branch, compose_path, project_name, project_id, auto_sync, sync_interval, last_sync_at, last_sync_status, last_sync_error, last_sync_commit, created_at, updated_at
FROM gitops_syncs
WHERE auto_sync = true;

-- name: SaveGitOpsSync :one
UPDATE gitops_syncs
SET
    name = ?,
    environment_id = ?,
    repository_id = ?,
    branch = ?,
    compose_path = ?,
    project_name = ?,
    project_id = ?,
    auto_sync = ?,
    sync_interval = ?,
    last_sync_at = ?,
    last_sync_status = ?,
    last_sync_error = ?,
    last_sync_commit = ?,
    updated_at = ?
WHERE id = ?
RETURNING id, name, environment_id, repository_id, branch, compose_path, project_name, project_id, auto_sync, sync_interval, last_sync_at, last_sync_status, last_sync_error, last_sync_commit, created_at, updated_at;

-- name: DeleteGitOpsSyncByID :execrows
DELETE FROM gitops_syncs
WHERE id = ?;

-- name: UpdateGitOpsSyncInterval :exec
UPDATE gitops_syncs
SET sync_interval = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateGitOpsSyncProjectID :exec
UPDATE gitops_syncs
SET project_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateGitOpsSyncStatus :exec
UPDATE gitops_syncs
SET
    last_sync_at = ?,
    last_sync_status = ?,
    last_sync_error = ?,
    last_sync_commit = ?,
    updated_at = ?
WHERE id = ?;

-- name: SetProjectGitOpsManagedBy :exec
UPDATE projects
SET gitops_managed_by = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ClearProjectGitOpsManagedByIfMatches :exec
UPDATE projects
SET gitops_managed_by = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
  AND gitops_managed_by = ?;
