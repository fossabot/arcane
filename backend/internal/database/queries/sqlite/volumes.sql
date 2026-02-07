-- name: CreateVolumeBackup :one
INSERT INTO volume_backups (
    id,
    volume_name,
    size,
    created_at,
    updated_at
)
VALUES (?, ?, ?, ?, ?)
RETURNING id, volume_name, size, created_at, updated_at;

-- name: ListVolumeBackupsByVolumeName :many
SELECT id, volume_name, size, created_at, updated_at
FROM volume_backups
WHERE volume_name = ?;

-- name: GetVolumeBackupByID :one
SELECT id, volume_name, size, created_at, updated_at
FROM volume_backups
WHERE id = ?
LIMIT 1;

-- name: DeleteVolumeBackupByID :execrows
DELETE FROM volume_backups
WHERE id = ?;
