package brewers

import "fmt"

type ErrBrewerAlreadyExists struct {
	Name string
}

func (e ErrBrewerAlreadyExists) Error() string {
	return fmt.Sprintf("brewer with name %s already exists", e.Name)
}

type ErrBrewerNotFound struct {
	ID int64
}

func (e ErrBrewerNotFound) Error() string {
	return fmt.Sprintf("brewer with id %d not found", e.ID)
}
