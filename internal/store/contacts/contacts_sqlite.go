// Description: This file contains the contactStore struct and its methods
// that are used to store and retrieve contacts from the store.
// This is for example purposes only and just uses an in-memory Map
package contacts

import (
	"beer_oclock/internal/db"
	"beer_oclock/internal/store"
	"context"
	"log"

	"modernc.org/sqlite"
)

type ContactStore struct {
	queries *db.Queries
	logger  *log.Logger
}

func NewContactStore(queries *db.Queries, logger *log.Logger) *ContactStore {
	return &ContactStore{
		logger:  logger,
		queries: queries,
	}
}

func (cs *ContactStore) AddContact(ctx context.Context, params db.AddContactParams) (db.Contact, error) {
	zero := db.Contact{}

	if params.Email == "" {
		return zero, store.ErrMissingField{Field: "email"}
	}

	if params.Name == "" {
		return zero, store.ErrMissingField{Field: "name"}
	}

	contact, err := cs.queries.AddContact(ctx, params)
	if err != nil {
		if sqlErr, ok := err.(*sqlite.Error); ok {
			if sqlErr.Code() == 2067 { // UNIQUE constraint failed
				return zero, ErrContactAlreadyExists{Email: params.Email}
			}
			cs.logger.Printf("error adding contact: %v, %v", err, sqlErr)
		}
		cs.logger.Printf("error adding contact: %v", err)
		return zero, err
	}

	cs.logger.Printf("contact added: %v", contact)
	return contact, nil
}

func (cs *ContactStore) GetContacts(ctx context.Context) ([]db.Contact, error) {
	contacts, err := cs.queries.GetContacts(ctx)
	if err != nil {
		cs.logger.Printf("error getting contacts: %v", err)
		return nil, err
	}
	return contacts, nil
}

func (cs *ContactStore) DeleteContact(ctx context.Context, id int64) (db.Contact, error) {
	contact, err := cs.queries.DeleteContact(ctx, id)
	if err != nil {
		cs.logger.Printf("error deleting contact: %v", err)
		return db.Contact{}, err
	}
	return contact, nil
}

func (cs *ContactStore) CountContacts(ctx context.Context) (int64, error) {
	count, err := cs.queries.CountContacts(ctx)
	if err != nil {
		cs.logger.Printf("error counting contacts: %v", err)
		return 0, err
	}
	return count, nil
}
