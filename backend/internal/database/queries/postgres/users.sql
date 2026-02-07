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
    $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13, $14, $15, $16
)
RETURNING id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change;

-- name: GetUserByUsername :one
SELECT id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change
FROM users
WHERE username = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change
FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByIDForUpdate :one
SELECT id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change
FROM users
WHERE id = $1
LIMIT 1
FOR UPDATE;

-- name: GetUserByOidcSubjectID :one
SELECT id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change
FROM users
WHERE oidc_subject_id = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change
FROM users
WHERE email = $1
LIMIT 1;

-- name: UpdateUser :one
UPDATE users
SET
    username = $2,
    password_hash = $3,
    display_name = $4,
    email = $5,
    roles = $6,
    require_password_change = $7,
    oidc_subject_id = $8,
    last_login = $9,
    updated_at = $10,
    oidc_access_token = $11,
    oidc_refresh_token = $12,
    oidc_access_token_expires_at = $13,
    locale = $14,
    requires_password_change = $15
WHERE id = $1
RETURNING id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: ListUsers :many
SELECT id, username, password_hash, display_name, email, roles, require_password_change, oidc_subject_id, last_login, created_at, updated_at, oidc_access_token, oidc_refresh_token, oidc_access_token_expires_at, locale, requires_password_change
FROM users;

-- name: DeleteUserByID :execrows
DELETE FROM users
WHERE id = $1;

-- name: UpdateUserPasswordHash :exec
UPDATE users
SET
    password_hash = $2,
    updated_at = $3
WHERE id = $1;
