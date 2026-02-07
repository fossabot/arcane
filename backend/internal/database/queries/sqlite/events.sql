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
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, type, severity, title, description, resource_type, resource_id, resource_name, user_id, username, environment_id, metadata, timestamp, created_at, updated_at;

-- name: ListEvents :many
SELECT id, type, severity, title, description, resource_type, resource_id, resource_name, user_id, username, environment_id, metadata, timestamp, created_at, updated_at
FROM events;

-- name: DeleteEventByID :execrows
DELETE FROM events
WHERE id = ?;

-- name: DeleteEventsOlderThan :execrows
DELETE FROM events
WHERE timestamp < ?;
