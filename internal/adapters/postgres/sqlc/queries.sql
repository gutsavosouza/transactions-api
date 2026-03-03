-- name: CreateUser :one
INSERT INTO users (id, cpf, name, password) 
VALUES($1, $2, $3, $4)
RETURNING *;

-- name: FindUserByCPF :one
SELECT *
FROM users
WHERE cpf = $1;

-- name: CreateAccount :one
INSERT INTO accounts(id, user_id)
VALUES($1, $2)
RETURNING *;

-- name: GetAccount :one
SELECT *
FROM accounts
WHERE id = $1;

-- name: ListUserAccounts :many
SELECT *
FROM accounts
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateBalance :one
UPDATE accounts
SET balance = $2, updated_at = now() 
WHERE id = $1
RETURNING *;

-- name: CreateTransaction :one
INSERT INTO transactions(id, user_id, from_account_id, to_account_id, amount, status)
VALUES($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUserTransactions :many
SELECT *
FROM transactions
WHERE user_id = $1
LIMIT $2 OFFSET $3;

-- name: UpdateTransactionsStatus :one
UPDATE transactions
SET status = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: GetTransaction :one
SELECT id, user_id, from_account_id, to_account_id, amount, status, created_at, updated_at
FROM transactions
WHERE id = $1;