// package repository provides data access and error types
package repository

import (
	"fmt"
)

// ErrTodoNotFound is returned when a todo with the specified ID does not exist
type ErrTodoNotFound struct {
	ID string
}

// Error implements the error interface
func (e ErrTodoNotFound) Error() string {
	return fmt.Sprintf("todo with id %s not found", e.ID)
}
