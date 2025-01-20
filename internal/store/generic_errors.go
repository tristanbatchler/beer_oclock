package store

import "fmt"

type ErrMissingField struct {
	Field string
}

func (e ErrMissingField) Error() string {
	return fmt.Sprintf("%s is required", e.Field)
}
