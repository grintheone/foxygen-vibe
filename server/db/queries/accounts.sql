-- name: CreateAccount :one
INSERT INTO accounts (username, password_hash)
VALUES ($1, $2)
RETURNING user_id, username, disabled, password_hash;
