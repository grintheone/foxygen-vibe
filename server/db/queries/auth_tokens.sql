-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING token_id, user_id, token_hash, expires_at, created_at, rotated_at, replaced_by, revoked_at;

-- name: GetRefreshTokenByHash :one
SELECT token_id, user_id, token_hash, expires_at, created_at, rotated_at, replaced_by, revoked_at
FROM refresh_tokens
WHERE token_hash = $1;

-- name: RotateRefreshToken :execrows
UPDATE refresh_tokens
SET rotated_at = NOW(), replaced_by = $2
WHERE token_id = $1
  AND rotated_at IS NULL
  AND revoked_at IS NULL
  AND expires_at > NOW();
