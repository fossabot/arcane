-- name: CreateNotificationLog :exec
INSERT INTO notification_logs (provider, image_ref, status, error, metadata, sent_at)
VALUES (?, ?, ?, ?, ?, ?);
