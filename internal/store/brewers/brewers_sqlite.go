package brewers

import (
	"beer_oclock/internal/db"
	"beer_oclock/internal/store"
	"context"
	"database/sql"
	"log"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type BrewerStore struct {
	queries *db.Queries
	logger  *log.Logger
}

func NewBrewerStore(queries *db.Queries, logger *log.Logger) *BrewerStore {
	return &BrewerStore{
		logger:  logger,
		queries: queries,
	}
}

func (bs *BrewerStore) AddBrewer(ctx context.Context, params db.AddBrewerParams) (db.Brewer, error) {
	zero := db.Brewer{}

	if params.Name == "" {
		return zero, store.ErrMissingField{Field: "name"}
	}

	brewer, err := bs.queries.AddBrewer(ctx, params)
	if err != nil {
		if sqlErr, ok := err.(*sqlite.Error); ok {
			if sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
				return zero, ErrBrewerAlreadyExists{Name: params.Name}
			}
		}
		bs.logger.Printf("error adding brewer: %v", err)
		return zero, err
	}

	bs.logger.Printf("brewer added: %v", brewer)
	return brewer, nil
}

func (bs *BrewerStore) GetBrewer(ctx context.Context, id int64) (db.Brewer, error) {
	brewer, err := bs.queries.GetBrewerById(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return db.Brewer{}, store.ErrBrewerNotFound{ID: id}
		}
		bs.logger.Printf("error getting brewer: %v", err)
		return db.Brewer{}, err
	}
	return brewer, nil
}

func (bs *BrewerStore) GetBrewers(ctx context.Context) ([]db.Brewer, error) {
	brewers, err := bs.queries.GetBrewers(ctx)
	if err != nil {
		bs.logger.Printf("error getting brewers: %v", err)
		return nil, err
	}
	return brewers, nil
}

func (bs *BrewerStore) DeleteBrewer(ctx context.Context, id int64) (db.Brewer, error) {
	zero := db.Brewer{}

	brewer, err := bs.queries.DeleteBrewer(ctx, id)
	if err != nil {
		if sqlErr, ok := err.(*sqlite.Error); ok {
			if sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY {
				return zero, store.ErrBrewerNotFound{ID: id}
			}
		}
		bs.logger.Printf("error deleting brewer: %v", err)
		return zero, err
	}

	bs.logger.Printf("brewer deleted: %v", brewer)
	return brewer, nil
}

func (bs *BrewerStore) CountBrewers(ctx context.Context) (int64, error) {
	count, err := bs.queries.CountBrewers(ctx)
	if err != nil {
		bs.logger.Printf("error counting brewers: %v", err)
		return 0, err
	}
	return count, nil
}
