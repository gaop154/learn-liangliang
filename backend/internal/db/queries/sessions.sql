-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at;

-- name: GetActiveSessionByTokenHash :one
SELECT s.id, s.user_id, s.token_hash, s.expires_at, s.revoked_at, s.created_at,
       u.username, COALESCE(u.display_name, '') AS display_name, u.is_active
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1
  AND s.revoked_at IS NULL
  AND s.expires_at > NOW()
  AND u.is_active = TRUE;

-- name: RevokeSession :exec
UPDATE sessions
SET revoked_at = NOW()
WHERE token_hash = $1 AND revoked_at IS NULL;
