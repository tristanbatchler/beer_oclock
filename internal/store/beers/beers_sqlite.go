package beers

import (
	"beer_oclock/internal/db"
	"beer_oclock/internal/store"
	"context"
	"database/sql"
	"log"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type BeerStore struct {
	queries *db.Queries
	logger  *log.Logger
}

func NewBeerStore(queries *db.Queries, logger *log.Logger) *BeerStore {
	return &BeerStore{
		logger:  logger,
		queries: queries,
	}
}

func (bs *BeerStore) AddBeer(ctx context.Context, params db.AddBeerParams) (db.Beer, error) {
	zero := db.Beer{}

	if params.Name == "" {
		return zero, store.ErrMissingField{Field: "name"}
	}
	if params.Abv < 0 {
		return zero, store.ErrInvalidField{Field: "abv", Reason: "must be >= 0"}
	}
	if !params.Rating.Valid {
		return zero, store.ErrMissingField{Field: "rating"}
	} else if params.Rating.Float64 < 0 || params.Rating.Float64 > 10 {
		return zero, store.ErrInvalidField{Field: "rating", Reason: "must be between 0 and 10"}
	}

	beer, err := bs.queries.AddBeer(ctx, params)
	if err != nil {
		if sqlErr, ok := err.(*sqlite.Error); ok {
			switch sqlErr.Code() {
			case sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY:
				return zero, store.ErrBrewerNotFound{ID: params.BrewerID.Int64}
			case sqlite3.SQLITE_CONSTRAINT_UNIQUE:
				return zero, store.ErrBeerAlreadyExists{Name: params.Name}
			}
		}
		bs.logger.Printf("error adding beer: %v", err)
		return zero, err
	}

	bs.logger.Printf("beer added: %v", beer)
	return beer, nil
}

func (bs *BeerStore) GetBeer(ctx context.Context, id int64) (db.Beer, error) {
	beer, err := bs.queries.GetBeerById(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return db.Beer{}, ErrBeerNotFound{ID: id}
		}
		bs.logger.Printf("error getting beer: %v", err)
		return db.Beer{}, err
	}
	return beer, nil
}

func (bs *BeerStore) GetBeers(ctx context.Context) ([]db.Beer, error) {
	beers, err := bs.queries.GetBeers(ctx)
	if err != nil {
		bs.logger.Printf("error getting beers: %v", err)
		return nil, err
	}
	return beers, nil
}

func (bs *BeerStore) DeleteBeer(ctx context.Context, id int64) (db.Beer, error) {
	zero := db.Beer{}

	beer, err := bs.queries.DeleteBeer(ctx, id)
	if err != nil {
		if sqlErr, ok := err.(*sqlite.Error); ok {
			if sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY {
				return zero, ErrBeerNotFound{ID: id}
			}
		}
		bs.logger.Printf("error deleting beer: %v", err)
		return zero, err
	}

	bs.logger.Printf("beer deleted: %v", beer)
	return beer, nil
}

func (bs *BeerStore) CountBeers(ctx context.Context) (int64, error) {
	count, err := bs.queries.CountBeers(ctx)
	if err != nil {
		bs.logger.Printf("error counting beers: %v", err)
		return 0, err
	}
	return count, nil
}

func (bs *BeerStore) UpdateBeer(ctx context.Context, params db.UpdateBeerParams) (db.Beer, error) {
	zero := db.Beer{}

	if params.Name.String == "" {
		return zero, store.ErrMissingField{Field: "name"}
	}
	if params.Rating.Float64 < 0 || params.Rating.Float64 > 10 {
		return zero, store.ErrInvalidField{Field: "rating", Reason: "must be between 0 and 10"}
	}

	beer, err := bs.queries.UpdateBeer(ctx, params)
	if err != nil {
		if sqlErr, ok := err.(*sqlite.Error); ok {
			switch sqlErr.Code() {
			case sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY:
				return zero, store.ErrBrewerNotFound{ID: params.BrewerID.Int64}
			case sqlite3.SQLITE_CONSTRAINT_UNIQUE:
				return zero, store.ErrBeerAlreadyExists{Name: params.Name.String}
			}
		}
		bs.logger.Printf("error updating beer: %v", err)
		return zero, err
	}

	bs.logger.Printf("beer updated: %v", beer)
	return beer, nil
}

func (bs *BeerStore) SearchBeers(ctx context.Context, query sql.NullString) ([]db.Beer, error) {
	beers, err := bs.queries.SearchBeers(ctx, query)
	if err != nil {
		bs.logger.Printf("error searching beers: %v", err)
		return nil, err
	}
	return beers, nil
}
