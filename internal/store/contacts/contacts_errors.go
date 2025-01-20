package contacts

import "fmt"

type ErrContactAlreadyExists struct {
	Email string
}

func (e ErrContactAlreadyExists) Error() string {
	return fmt.Sprintf("contact with email %s already exists", e.Email)
}

type ErrContactNotFound struct {
	ID int64
}

func (e ErrContactNotFound) Error() string {
	return fmt.Sprintf("contact with id %d not found", e.ID)
}
