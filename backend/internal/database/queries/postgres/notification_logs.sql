-- name: CreateNotificationLog :exec
INSERT INTO notification_logs (provider, image_ref, status, error, metadata, sent_at)
VALUES ($1, $2, $3, $4, $5, $6);
