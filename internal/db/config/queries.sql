-- name: AddContact :one
INSERT INTO contacts (name, email) 
VALUES (?, ?)
RETURNING *;

-- name: GetContactById :one
SELECT * 
FROM contacts
WHERE id = ?;

-- name: GetContacts :many
SELECT *
FROM contacts;

-- name: DeleteContact :one
DELETE FROM contacts
WHERE id = ?
RETURNING *;

-- name: CountContacts :one
SELECT COUNT(*)
FROM contacts;