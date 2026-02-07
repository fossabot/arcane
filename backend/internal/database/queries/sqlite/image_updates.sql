-- name: GetImageUpdateByID :one
SELECT id,
			 repository,
			 tag,
			 has_update,
			 update_type,
			 current_version,
			 latest_version,
			 current_digest,
			 latest_digest,
			 check_time,
			 response_time_ms,
			 last_error,
			 auth_method,
			 auth_username,
			 auth_registry,
			 used_credential,
			 notification_sent,
			 created_at,
			 updated_at
FROM image_updates
WHERE id = ?
LIMIT 1;

-- name: SaveImageUpdate :one
INSERT INTO image_updates (
		id,
		repository,
		tag,
		has_update,
		update_type,
		current_version,
		latest_version,
		current_digest,
		latest_digest,
		check_time,
		response_time_ms,
		last_error,
		auth_method,
		auth_username,
		auth_registry,
		used_credential,
		notification_sent,
		created_at,
		updated_at
)
VALUES (
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		CURRENT_TIMESTAMP,
		CURRENT_TIMESTAMP
)
ON CONFLICT(id) DO UPDATE SET
		repository = excluded.repository,
		tag = excluded.tag,
		has_update = excluded.has_update,
		update_type = excluded.update_type,
		current_version = excluded.current_version,
		latest_version = excluded.latest_version,
		current_digest = excluded.current_digest,
		latest_digest = excluded.latest_digest,
		check_time = excluded.check_time,
		response_time_ms = excluded.response_time_ms,
		last_error = excluded.last_error,
		auth_method = excluded.auth_method,
		auth_username = excluded.auth_username,
		auth_registry = excluded.auth_registry,
		used_credential = excluded.used_credential,
		notification_sent = excluded.notification_sent,
		updated_at = CURRENT_TIMESTAMP
RETURNING id,
					repository,
					tag,
					has_update,
					update_type,
					current_version,
					latest_version,
					current_digest,
					latest_digest,
					check_time,
					response_time_ms,
					last_error,
					auth_method,
					auth_username,
					auth_registry,
					used_credential,
					notification_sent,
					created_at,
					updated_at;

-- name: ListImageUpdates :many
SELECT id,
			 repository,
			 tag,
			 has_update,
			 update_type,
			 current_version,
			 latest_version,
			 current_digest,
			 latest_digest,
			 check_time,
			 response_time_ms,
			 last_error,
			 auth_method,
			 auth_username,
			 auth_registry,
			 used_credential,
			 notification_sent,
			 created_at,
			 updated_at
FROM image_updates;

-- name: ListImageUpdatesByIDs :many
SELECT id,
			 repository,
			 tag,
			 has_update,
			 update_type,
			 current_version,
			 latest_version,
			 current_digest,
			 latest_digest,
			 check_time,
			 response_time_ms,
			 last_error,
			 auth_method,
			 auth_username,
			 auth_registry,
			 used_credential,
			 notification_sent,
			 created_at,
			 updated_at
FROM image_updates
WHERE id IN (sqlc.slice('ids'));

-- name: ListImageUpdatesWithUpdate :many
SELECT id,
			 repository,
			 tag,
			 has_update,
			 update_type,
			 current_version,
			 latest_version,
			 current_digest,
			 latest_digest,
			 check_time,
			 response_time_ms,
			 last_error,
			 auth_method,
			 auth_username,
			 auth_registry,
			 used_credential,
			 notification_sent,
			 created_at,
			 updated_at
FROM image_updates
WHERE has_update = true;

-- name: ListUnnotifiedImageUpdates :many
SELECT id,
			 repository,
			 tag,
			 has_update,
			 update_type,
			 current_version,
			 latest_version,
			 current_digest,
			 latest_digest,
			 check_time,
			 response_time_ms,
			 last_error,
			 auth_method,
			 auth_username,
			 auth_registry,
			 used_credential,
			 notification_sent,
			 created_at,
			 updated_at
FROM image_updates
WHERE has_update = true
	AND notification_sent = false;

-- name: MarkImageUpdatesNotified :exec
UPDATE image_updates
SET notification_sent = true,
		updated_at = CURRENT_TIMESTAMP
WHERE id IN (sqlc.slice('ids'));

-- name: DeleteImageUpdatesByIDs :execrows
DELETE FROM image_updates
WHERE id IN (sqlc.slice('ids'));

-- name: CountImageUpdates :one
SELECT COUNT(*)
FROM image_updates;

-- name: CountImageUpdatesWithUpdate :one
SELECT COUNT(*)
FROM image_updates
WHERE has_update = true;

-- name: CountImageUpdatesWithUpdateType :one
SELECT COUNT(*)
FROM image_updates
WHERE has_update = true
	AND update_type = ?;

-- name: CountImageUpdatesWithErrors :one
SELECT COUNT(*)
FROM image_updates
WHERE last_error IS NOT NULL;

-- name: UpdateImageUpdateHasUpdateByRepositoryTag :exec
UPDATE image_updates
SET has_update = ?,
		updated_at = CURRENT_TIMESTAMP
WHERE repository = ?
	AND tag = ?;
