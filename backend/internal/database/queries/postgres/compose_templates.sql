-- name: GetComposeTemplateByID :one
SELECT id, name, description, content, env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at
FROM compose_templates
WHERE id = $1
LIMIT 1;

-- name: ListComposeTemplates :many
SELECT id, name, description, content, env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at
FROM compose_templates;

-- name: ListComposeTemplatesLite :many
SELECT id, name, description, ''::text AS content, NULL::text AS env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at
FROM compose_templates;

-- name: FindLocalComposeTemplateByDescriptionOrName :one
SELECT id, name, description, content, env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at
FROM compose_templates
WHERE is_remote = false
  AND registry_id IS NULL
  AND (description = $1 OR name = $2)
LIMIT 1;

-- name: FindLocalComposeTemplateByDescription :one
SELECT id, name, description, content, env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at
FROM compose_templates
WHERE is_remote = false
  AND registry_id IS NULL
  AND description = $1
LIMIT 1;

-- name: CreateComposeTemplate :one
INSERT INTO compose_templates (
    id,
    name,
    description,
    content,
    env_content,
    is_custom,
    is_remote,
    registry_id,
    meta_version,
    meta_author,
    meta_tags,
    meta_remote_url,
    meta_env_url,
    meta_documentation_url,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING id, name, description, content, env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at;

-- name: SaveComposeTemplate :one
UPDATE compose_templates
SET
    name = $1,
    description = $2,
    content = $3,
    env_content = $4,
    is_custom = $5,
    is_remote = $6,
    registry_id = $7,
    meta_version = $8,
    meta_author = $9,
    meta_tags = $10,
    meta_remote_url = $11,
    meta_env_url = $12,
    meta_documentation_url = $13,
    updated_at = $14
WHERE id = $15
RETURNING id, name, description, content, env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at;

-- name: DeleteComposeTemplateByID :execrows
DELETE FROM compose_templates
WHERE id = $1;
