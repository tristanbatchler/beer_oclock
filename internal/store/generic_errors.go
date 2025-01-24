package store

import "fmt"

type ErrMissingField struct {
	Field string
}

func (e ErrMissingField) Error() string {
	return fmt.Sprintf("%s is required", e.Field)
}

type ErrInvalidField struct {
	Field  string
	Reason string
}

func (e ErrInvalidField) Error() string {
	return fmt.Sprintf("%s is invalid: %s", e.Field, e.Reason)
}

type ErrBrewerNotFound struct {
	ID int64
}

func (e ErrBrewerNotFound) Error() string {
	return fmt.Sprintf("brewer with id %d not found", e.ID)
}

type ErrBeerAlreadyExists struct {
	BrewerId int64
	Name     string
}

func (e ErrBeerAlreadyExists) Error() string {
	return fmt.Sprintf("beer with name %s already exists for brewer with id %d", e.Name, e.BrewerId)
}
