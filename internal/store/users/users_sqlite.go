package users

import (
	"beer_oclock/internal/db"
	"beer_oclock/internal/store"
	"context"
	"database/sql"
	"log"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type UserStore struct {
	queries *db.Queries
	logger  *log.Logger
}

func NewUserStore(queries *db.Queries, logger *log.Logger) *UserStore {
	return &UserStore{
		logger:  logger,
		queries: queries,
	}
}

func (cs *UserStore) AddUser(ctx context.Context, params db.AddUserParams) (db.User, error) {
	zero := db.User{}

	if params.Username == "" {
		return zero, store.ErrMissingField{Field: "username"}
	}

	user, err := cs.queries.AddUser(ctx, params)
	if err != nil {
		if sqlErr, ok := err.(*sqlite.Error); ok {
			if sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
				return zero, ErrUserAlreadyExists{Username: params.Username}
			}
			cs.logger.Printf("error adding user: %v, %v", err, sqlErr)
		}
		cs.logger.Printf("error adding user: %v", err)
		return zero, err
	}

	cs.logger.Printf("user added: %v", user)
	return user, nil
}

func (cs *UserStore) GetUsers(ctx context.Context) ([]db.User, error) {
	users, err := cs.queries.GetUsers(ctx)
	if err != nil {
		cs.logger.Printf("error getting users: %v", err)
		return nil, err
	}
	return users, nil
}

func (cs *UserStore) GetUserById(ctx context.Context, id int64) (db.User, error) {
	zero := db.User{}

	user, err := cs.queries.GetUserById(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return zero, ErrUserNotFound{ID: id}
		}
		cs.logger.Printf("error getting user by id: %v", err)
		return zero, err
	}

	return user, nil
}

func (cs *UserStore) GetUserByUsername(ctx context.Context, username string) (db.User, error) {
	zero := db.User{}

	user, err := cs.queries.GetUserByUsername(ctx, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return zero, ErrUserNotFound{Username: username}
		}
		cs.logger.Printf("error getting user by username: %v", err)
		return zero, err
	}

	return user, nil
}

func (cs *UserStore) DeleteUser(ctx context.Context, id int64) (db.User, error) {
	zero := db.User{}

	user, err := cs.queries.DeleteUser(ctx, id)
	if err != nil {
		if sqlErr, ok := err.(*sqlite.Error); ok {
			if sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY {
				return zero, ErrUserNotFound{ID: id}
			}
		}
		cs.logger.Printf("error deleting user: %v", err)
		return zero, err
	}

	cs.logger.Printf("user deleted: %v", user)
	return user, nil
}

func (cs *UserStore) CountUsers(ctx context.Context) (int64, error) {
	count, err := cs.queries.CountUsers(ctx)
	if err != nil {
		cs.logger.Printf("error counting users: %v", err)
		return 0, err
	}
	return count, nil
}

func (cs *UserStore) SetUserLastLogin(ctx context.Context, id int64) error {
	err := cs.queries.SetUserLastLogin(ctx, id)
	if err != nil {
		cs.logger.Printf("error setting user last login: %v", err)
		return err
	}
	return nil
}
