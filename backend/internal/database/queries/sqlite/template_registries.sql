-- name: GetTemplateRegistryByID :one
SELECT id, name, url, enabled, description, created_at, updated_at
FROM template_registries
WHERE id = ?
LIMIT 1;

-- name: ListTemplateRegistries :many
SELECT id, name, url, enabled, description, created_at, updated_at
FROM template_registries;

-- name: CreateTemplateRegistry :one
INSERT INTO template_registries (
    id,
    name,
    url,
    enabled,
    description,
    created_at,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, url, enabled, description, created_at, updated_at;

-- name: SaveTemplateRegistry :one
UPDATE template_registries
SET
    name = ?,
    url = ?,
    enabled = ?,
    description = ?,
    updated_at = ?
WHERE id = ?
RETURNING id, name, url, enabled, description, created_at, updated_at;

-- name: DeleteTemplateRegistryByID :execrows
DELETE FROM template_registries
WHERE id = ?;
