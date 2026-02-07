-- name: CreateProject :one
INSERT INTO projects (
    id,
    name,
    dir_name,
    path,
    status,
    service_count,
    running_count,
    status_reason,
    gitops_managed_by,
    created_at,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, dir_name, path, status, service_count, running_count, auto_update, is_external, is_legacy, is_remote, created_at, updated_at, status_reason, gitops_managed_by;

-- name: GetProjectByID :one
SELECT id, name, dir_name, path, status, service_count, running_count, auto_update, is_external, is_legacy, is_remote, created_at, updated_at, status_reason, gitops_managed_by
FROM projects
WHERE id = ?
LIMIT 1;

-- name: GetProjectByPathOrDir :one
SELECT id, name, dir_name, path, status, service_count, running_count, auto_update, is_external, is_legacy, is_remote, created_at, updated_at, status_reason, gitops_managed_by
FROM projects
WHERE path = ? OR dir_name = ?
LIMIT 1;

-- name: ListProjects :many
SELECT id, name, dir_name, path, status, service_count, running_count, auto_update, is_external, is_legacy, is_remote, created_at, updated_at, status_reason, gitops_managed_by
FROM projects;

-- name: SaveProject :one
UPDATE projects
SET
    name = ?,
    dir_name = ?,
    path = ?,
    status = ?,
    service_count = ?,
    running_count = ?,
    status_reason = ?,
    gitops_managed_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING id, name, dir_name, path, status, service_count, running_count, auto_update, is_external, is_legacy, is_remote, created_at, updated_at, status_reason, gitops_managed_by;

-- name: DeleteProjectByID :execrows
DELETE FROM projects
WHERE id = ?;

-- name: UpdateProjectStatus :exec
UPDATE projects
SET status = ?,
    updated_at = ?
WHERE id = ?;

-- name: UpdateProjectStatusAndCounts :exec
UPDATE projects
SET status = ?,
    service_count = ?,
    running_count = ?,
    updated_at = ?
WHERE id = ?;

-- name: UpdateProjectServiceCount :exec
UPDATE projects
SET service_count = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;
