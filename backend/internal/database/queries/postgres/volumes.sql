-- name: CreateVolumeBackup :one
INSERT INTO volume_backups (
    id,
    volume_name,
    size,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, volume_name, size, created_at, updated_at;

-- name: ListVolumeBackupsByVolumeName :many
SELECT id, volume_name, size, created_at, updated_at
FROM volume_backups
WHERE volume_name = $1;

-- name: GetVolumeBackupByID :one
SELECT id, volume_name, size, created_at, updated_at
FROM volume_backups
WHERE id = $1
LIMIT 1;

-- name: DeleteVolumeBackupByID :execrows
DELETE FROM volume_backups
WHERE id = $1;
