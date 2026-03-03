-- name: CreateAccount :one
INSERT INTO accounts (username, password_hash)
VALUES ($1, $2)
RETURNING user_id, username, disabled, password_hash;

-- name: CreateUserProfile :one
INSERT INTO users (user_id)
VALUES ($1)
RETURNING user_id, first_name, last_name, department, email, phone, logo, latest_ticket;

-- name: GetAccountByUsername :one
SELECT user_id, username, disabled, password_hash
FROM accounts
WHERE username = $1;

-- name: GetAccountByUserID :one
SELECT user_id, username, disabled, password_hash
FROM accounts
WHERE user_id = $1;

-- name: GetUserProfileByUserID :one
SELECT
  a.user_id,
  a.username,
  TRIM(CONCAT(u.first_name, ' ', u.last_name)) AS name,
  u.email,
  COALESCE(d.title, '') AS department
FROM accounts AS a
JOIN users AS u ON u.user_id = a.user_id
LEFT JOIN departments AS d ON d.id = u.department
WHERE a.user_id = $1;
