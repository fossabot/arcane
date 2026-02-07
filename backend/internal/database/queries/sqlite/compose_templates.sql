-- name: GetComposeTemplateByID :one
SELECT id, name, description, content, env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at
FROM compose_templates
WHERE id = ?
LIMIT 1;

-- name: ListComposeTemplates :many
SELECT id, name, description, content, env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at
FROM compose_templates;

-- name: ListComposeTemplatesLite :many
SELECT id, name, description, '' AS content, NULL AS env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at
FROM compose_templates;

-- name: FindLocalComposeTemplateByDescriptionOrName :one
SELECT id, name, description, content, env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at
FROM compose_templates
WHERE is_remote = false
  AND registry_id IS NULL
  AND (description = ? OR name = ?)
LIMIT 1;

-- name: FindLocalComposeTemplateByDescription :one
SELECT id, name, description, content, env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at
FROM compose_templates
WHERE is_remote = false
  AND registry_id IS NULL
  AND description = ?
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
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, description, content, env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at;

-- name: SaveComposeTemplate :one
UPDATE compose_templates
SET
    name = ?,
    description = ?,
    content = ?,
    env_content = ?,
    is_custom = ?,
    is_remote = ?,
    registry_id = ?,
    meta_version = ?,
    meta_author = ?,
    meta_tags = ?,
    meta_remote_url = ?,
    meta_env_url = ?,
    meta_documentation_url = ?,
    updated_at = ?
WHERE id = ?
RETURNING id, name, description, content, env_content, is_custom, is_remote, registry_id, meta_version, meta_author, meta_tags, meta_remote_url, meta_env_url, meta_documentation_url, created_at, updated_at;

-- name: DeleteComposeTemplateByID :execrows
DELETE FROM compose_templates
WHERE id = ?;
