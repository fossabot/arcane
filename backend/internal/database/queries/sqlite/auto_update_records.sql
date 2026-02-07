-- name: CreateAutoUpdateRecord :exec
INSERT INTO auto_update_records (
    id,
    resource_id,
    resource_type,
    resource_name,
    status,
    start_time,
    end_time,
    update_available,
    update_applied,
    old_image_versions,
    new_image_versions,
    error,
    details,
    created_at,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

-- name: ListAutoUpdateRecords :many
SELECT id,
       resource_id,
       resource_type,
       resource_name,
       status,
       start_time,
       end_time,
       update_available,
       update_applied,
       old_image_versions,
       new_image_versions,
       error,
       details,
       created_at,
       updated_at
FROM auto_update_records
ORDER BY start_time DESC;

-- name: ListAutoUpdateRecordsLimited :many
SELECT id,
       resource_id,
       resource_type,
       resource_name,
       status,
       start_time,
       end_time,
       update_available,
       update_applied,
       old_image_versions,
       new_image_versions,
       error,
       details,
       created_at,
       updated_at
FROM auto_update_records
ORDER BY start_time DESC
LIMIT ?;
