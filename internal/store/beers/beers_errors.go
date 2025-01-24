package beers

import "fmt"

type ErrBeerNotFound struct {
	ID int64
}

func (e ErrBeerNotFound) Error() string {
	return fmt.Sprintf("beer with id %d not found", e.ID)
}
