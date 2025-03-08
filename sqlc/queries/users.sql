-- name: CreateUser :one
INSERT INTO users (email, password, name)
VALUES ($1, $2, $3)
RETURNING id, email, name, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password, name, created_at, updated_at
FROM users
WHERE email = $1;
