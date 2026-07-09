-- name: GetUserByUsername :one
SELECT id, username, password_hash, COALESCE(display_name, '') AS display_name, is_active, created_at, updated_at
FROM users
WHERE username = $1;

-- name: CreateUser :one
INSERT INTO users (username, password_hash, display_name)
VALUES ($1, $2, NULLIF(sqlc.arg(display_name)::text, ''))
ON CONFLICT (username) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    display_name = COALESCE(EXCLUDED.display_name, users.display_name),
    is_active = TRUE,
    updated_at = NOW()
RETURNING id, username, password_hash, COALESCE(display_name, '') AS display_name, is_active, created_at, updated_at;
