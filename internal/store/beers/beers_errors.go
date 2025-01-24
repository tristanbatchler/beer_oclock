package beers

import "fmt"

type ErrBeerAlreadyExists struct {
	BrewerId int64
	Name     string
}

func (e ErrBeerAlreadyExists) Error() string {
	return fmt.Sprintf("beer with name %s already exists for brewer with id %d", e.Name, e.BrewerId)
}

type ErrBeerNotFound struct {
	ID int64
}

func (e ErrBeerNotFound) Error() string {
	return fmt.Sprintf("beer with id %d not found", e.ID)
}
