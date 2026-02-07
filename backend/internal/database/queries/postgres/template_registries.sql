-- name: GetTemplateRegistryByID :one
SELECT id, name, url, enabled, description, created_at, updated_at
FROM template_registries
WHERE id = $1
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
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, url, enabled, description, created_at, updated_at;

-- name: SaveTemplateRegistry :one
UPDATE template_registries
SET
    name = $1,
    url = $2,
    enabled = $3,
    description = $4,
    updated_at = $5
WHERE id = $6
RETURNING id, name, url, enabled, description, created_at, updated_at;

-- name: DeleteTemplateRegistryByID :execrows
DELETE FROM template_registries
WHERE id = $1;
