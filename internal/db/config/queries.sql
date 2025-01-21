/* === CONTACTS === */

-- name: AddUser :one
INSERT INTO users (username, password_hash) 
VALUES (?, ?)
RETURNING *;

-- name: GetUserById :one
SELECT * 
FROM users
WHERE id = ?;

-- name: GetUserByUsername :one
SELECT *
FROM users
WHERE username = ?;

-- name: GetUsers :many
SELECT *
FROM users;

-- name: DeleteUser :one
DELETE FROM users
WHERE id = ?
RETURNING *;

-- name: CountUsers :one
SELECT COUNT(*)
FROM users;

-- name: SetUserLastLogin :exec
UPDATE users
SET last_login = datetime()
WHERE id = ?;

/* === BREWERS === */

-- name: AddBrewer :one
INSERT INTO brewers (name, location)
VALUES (?, ?)
RETURNING *;

-- name: GetBrewerById :one
SELECT *
FROM brewers
WHERE id = ?;

-- name: GetBrewers :many
SELECT *
FROM brewers;

-- name: DeleteBrewer :one
DELETE FROM brewers
WHERE id = ?
RETURNING *;

-- name: CountBrewers :one
SELECT COUNT(*)
FROM brewers;

/* === BEERS === */

-- name: AddBeer :one
INSERT INTO beers (name, brewer_id, style, abv)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetBeerById :one
SELECT *
FROM beers
WHERE id = ?;

-- name: GetBeers :many
SELECT *
FROM beers;

-- name: DeleteBeer :one
DELETE FROM beers
WHERE id = ?
RETURNING *;

-- name: CountBeers :one
SELECT COUNT(*)
FROM beers;
