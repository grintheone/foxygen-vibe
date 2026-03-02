-- name: CreateMessage :one
INSERT INTO messages (content)
VALUES ($1)
RETURNING id, content, created_at;

-- name: GetLatestMessage :one
SELECT id, content, created_at
FROM messages
ORDER BY id DESC
LIMIT 1;

-- name: ListMessages :many
SELECT id, content, created_at
FROM messages
ORDER BY id DESC
LIMIT $1;
