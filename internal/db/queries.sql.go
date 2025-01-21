// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: queries.sql

package db

import (
	"context"
	"database/sql"
)

const addBeer = `-- name: AddBeer :one

INSERT INTO beers (name, brewer_id, style, abv)
VALUES (?, ?, ?, ?)
RETURNING id, name, brewer_id, style, abv
`

type AddBeerParams struct {
	Name     string
	BrewerID sql.NullInt64
	Style    sql.NullString
	Abv      float64
}

// === BEERS ===
func (q *Queries) AddBeer(ctx context.Context, arg AddBeerParams) (Beer, error) {
	row := q.db.QueryRowContext(ctx, addBeer,
		arg.Name,
		arg.BrewerID,
		arg.Style,
		arg.Abv,
	)
	var i Beer
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.BrewerID,
		&i.Style,
		&i.Abv,
	)
	return i, err
}

const addBrewer = `-- name: AddBrewer :one

INSERT INTO brewers (name, location)
VALUES (?, ?)
RETURNING id, name, location
`

type AddBrewerParams struct {
	Name     string
	Location sql.NullString
}

// === BREWERS ===
func (q *Queries) AddBrewer(ctx context.Context, arg AddBrewerParams) (Brewer, error) {
	row := q.db.QueryRowContext(ctx, addBrewer, arg.Name, arg.Location)
	var i Brewer
	err := row.Scan(&i.ID, &i.Name, &i.Location)
	return i, err
}

const addUser = `-- name: AddUser :one

INSERT INTO users (username, password_hash) 
VALUES (?, ?)
RETURNING id, username, password_hash, created_at, last_login
`

type AddUserParams struct {
	Username     string
	PasswordHash string
}

// === CONTACTS ===
func (q *Queries) AddUser(ctx context.Context, arg AddUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, addUser, arg.Username, arg.PasswordHash)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.PasswordHash,
		&i.CreatedAt,
		&i.LastLogin,
	)
	return i, err
}

const countBeers = `-- name: CountBeers :one
SELECT COUNT(*)
FROM beers
`

func (q *Queries) CountBeers(ctx context.Context) (int64, error) {
	row := q.db.QueryRowContext(ctx, countBeers)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const countBrewers = `-- name: CountBrewers :one
SELECT COUNT(*)
FROM brewers
`

func (q *Queries) CountBrewers(ctx context.Context) (int64, error) {
	row := q.db.QueryRowContext(ctx, countBrewers)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const countUsers = `-- name: CountUsers :one
SELECT COUNT(*)
FROM users
`

func (q *Queries) CountUsers(ctx context.Context) (int64, error) {
	row := q.db.QueryRowContext(ctx, countUsers)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const deleteBeer = `-- name: DeleteBeer :one
DELETE FROM beers
WHERE id = ?
RETURNING id, name, brewer_id, style, abv
`

func (q *Queries) DeleteBeer(ctx context.Context, id int64) (Beer, error) {
	row := q.db.QueryRowContext(ctx, deleteBeer, id)
	var i Beer
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.BrewerID,
		&i.Style,
		&i.Abv,
	)
	return i, err
}

const deleteBrewer = `-- name: DeleteBrewer :one
DELETE FROM brewers
WHERE id = ?
RETURNING id, name, location
`

func (q *Queries) DeleteBrewer(ctx context.Context, id int64) (Brewer, error) {
	row := q.db.QueryRowContext(ctx, deleteBrewer, id)
	var i Brewer
	err := row.Scan(&i.ID, &i.Name, &i.Location)
	return i, err
}

const deleteUser = `-- name: DeleteUser :one
DELETE FROM users
WHERE id = ?
RETURNING id, username, password_hash, created_at, last_login
`

func (q *Queries) DeleteUser(ctx context.Context, id int64) (User, error) {
	row := q.db.QueryRowContext(ctx, deleteUser, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.PasswordHash,
		&i.CreatedAt,
		&i.LastLogin,
	)
	return i, err
}

const getBeerById = `-- name: GetBeerById :one
SELECT id, name, brewer_id, style, abv
FROM beers
WHERE id = ?
`

func (q *Queries) GetBeerById(ctx context.Context, id int64) (Beer, error) {
	row := q.db.QueryRowContext(ctx, getBeerById, id)
	var i Beer
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.BrewerID,
		&i.Style,
		&i.Abv,
	)
	return i, err
}

const getBeers = `-- name: GetBeers :many
SELECT id, name, brewer_id, style, abv
FROM beers
`

func (q *Queries) GetBeers(ctx context.Context) ([]Beer, error) {
	rows, err := q.db.QueryContext(ctx, getBeers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Beer
	for rows.Next() {
		var i Beer
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.BrewerID,
			&i.Style,
			&i.Abv,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getBrewerById = `-- name: GetBrewerById :one
SELECT id, name, location
FROM brewers
WHERE id = ?
`

func (q *Queries) GetBrewerById(ctx context.Context, id int64) (Brewer, error) {
	row := q.db.QueryRowContext(ctx, getBrewerById, id)
	var i Brewer
	err := row.Scan(&i.ID, &i.Name, &i.Location)
	return i, err
}

const getBrewers = `-- name: GetBrewers :many
SELECT id, name, location
FROM brewers
`

func (q *Queries) GetBrewers(ctx context.Context) ([]Brewer, error) {
	rows, err := q.db.QueryContext(ctx, getBrewers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Brewer
	for rows.Next() {
		var i Brewer
		if err := rows.Scan(&i.ID, &i.Name, &i.Location); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUserById = `-- name: GetUserById :one
SELECT id, username, password_hash, created_at, last_login 
FROM users
WHERE id = ?
`

func (q *Queries) GetUserById(ctx context.Context, id int64) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserById, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.PasswordHash,
		&i.CreatedAt,
		&i.LastLogin,
	)
	return i, err
}

const getUserByUsername = `-- name: GetUserByUsername :one
SELECT id, username, password_hash, created_at, last_login
FROM users
WHERE username = ?
`

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByUsername, username)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.PasswordHash,
		&i.CreatedAt,
		&i.LastLogin,
	)
	return i, err
}

const getUsers = `-- name: GetUsers :many
SELECT id, username, password_hash, created_at, last_login
FROM users
`

func (q *Queries) GetUsers(ctx context.Context) ([]User, error) {
	rows, err := q.db.QueryContext(ctx, getUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.Username,
			&i.PasswordHash,
			&i.CreatedAt,
			&i.LastLogin,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const setUserLastLogin = `-- name: SetUserLastLogin :exec
UPDATE users
SET last_login = datetime()
WHERE id = ?
`

func (q *Queries) SetUserLastLogin(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, setUserLastLogin, id)
	return err
}
