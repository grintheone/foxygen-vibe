-- name: CreateAccount :one
INSERT INTO accounts (username, password_hash)
VALUES ($1, $2)
RETURNING user_id, username, disabled, password_hash;

-- name: CreateUserProfile :one
INSERT INTO users (user_id)
VALUES ($1)
RETURNING user_id, first_name, last_name, department_id, email, phone, logo, latest_ticket;

-- name: GetAccountByUsername :one
SELECT user_id, username, disabled, password_hash
FROM accounts
WHERE username = $1;
