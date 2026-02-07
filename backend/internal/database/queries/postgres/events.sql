-- name: CreateEvent :one
INSERT INTO events (
    id,
    type,
    severity,
    title,
    description,
    resource_type,
    resource_id,
    resource_name,
    user_id,
    username,
    environment_id,
    metadata,
    timestamp,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING id, type, severity, title, description, resource_type, resource_id, resource_name, user_id, username, environment_id, metadata, timestamp, created_at, updated_at;

-- name: ListEvents :many
SELECT id, type, severity, title, description, resource_type, resource_id, resource_name, user_id, username, environment_id, metadata, timestamp, created_at, updated_at
FROM events;

-- name: DeleteEventByID :execrows
DELETE FROM events
WHERE id = $1;

-- name: DeleteEventsOlderThan :execrows
DELETE FROM events
WHERE timestamp < $1;
