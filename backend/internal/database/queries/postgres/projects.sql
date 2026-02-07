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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, name, dir_name, path, status, service_count, running_count, auto_update, is_external, is_legacy, is_remote, created_at, updated_at, status_reason, gitops_managed_by;

-- name: GetProjectByID :one
SELECT id, name, dir_name, path, status, service_count, running_count, auto_update, is_external, is_legacy, is_remote, created_at, updated_at, status_reason, gitops_managed_by
FROM projects
WHERE id = $1
LIMIT 1;

-- name: GetProjectByPathOrDir :one
SELECT id, name, dir_name, path, status, service_count, running_count, auto_update, is_external, is_legacy, is_remote, created_at, updated_at, status_reason, gitops_managed_by
FROM projects
WHERE path = $1 OR dir_name = $2
LIMIT 1;

-- name: ListProjects :many
SELECT id, name, dir_name, path, status, service_count, running_count, auto_update, is_external, is_legacy, is_remote, created_at, updated_at, status_reason, gitops_managed_by
FROM projects;

-- name: SaveProject :one
UPDATE projects
SET
    name = $1,
    dir_name = $2,
    path = $3,
    status = $4,
    service_count = $5,
    running_count = $6,
    status_reason = $7,
    gitops_managed_by = $8,
    updated_at = $9
WHERE id = $10
RETURNING id, name, dir_name, path, status, service_count, running_count, auto_update, is_external, is_legacy, is_remote, created_at, updated_at, status_reason, gitops_managed_by;

-- name: DeleteProjectByID :execrows
DELETE FROM projects
WHERE id = $1;

-- name: UpdateProjectStatus :exec
UPDATE projects
SET status = $1,
    updated_at = $2
WHERE id = $3;

-- name: UpdateProjectStatusAndCounts :exec
UPDATE projects
SET status = $1,
    service_count = $2,
    running_count = $3,
    updated_at = $4
WHERE id = $5;

-- name: UpdateProjectServiceCount :exec
UPDATE projects
SET service_count = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2;
