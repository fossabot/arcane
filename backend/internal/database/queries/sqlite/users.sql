-- name: CreateUser :one
INSERT INTO users (
    id,
    username,
    password_hash,
    display_name,
    email,
    roles,
    require_password_change,
    oidc_subject_id,
    last_login,
    created_at,
    updated_at,
    oidc_access_token,
    oidc_refresh_token,
    oidc_access_token_expires_at,
    locale,
    requires_password_change
)
VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change;

-- name: GetUserByUsername :one
SELECT id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change
FROM users
WHERE username = ?
LIMIT 1;

-- name: GetUserByID :one
SELECT id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change
FROM users
WHERE id = ?
LIMIT 1;

-- name: GetUserByIDForUpdate :one
SELECT id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change
FROM users
WHERE id = ?
LIMIT 1;

-- name: GetUserByOidcSubjectID :one
SELECT id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change
FROM users
WHERE oidc_subject_id = ?
LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change
FROM users
WHERE email = ?
LIMIT 1;

-- name: UpdateUser :one
UPDATE users
SET
    username = ?,
    password_hash = ?,
    display_name = ?,
    email = ?,
    roles = ?,
    require_password_change = ?,
    oidc_subject_id = ?,
    last_login = ?,
    updated_at = ?,
    oidc_access_token = ?,
    oidc_refresh_token = ?,
    oidc_access_token_expires_at = ?,
    locale = ?,
    requires_password_change = ?
WHERE id = ?
RETURNING id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: ListUsers :many
SELECT id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change
FROM users;

-- name: DeleteUserByID :execrows
DELETE FROM users
WHERE id = ?;

-- name: UpdateUserPasswordHash :exec
UPDATE users
SET
    password_hash = ?,
    updated_at = ?
WHERE id = ?;
