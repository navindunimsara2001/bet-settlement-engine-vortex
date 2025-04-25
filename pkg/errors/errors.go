package errors

import "fmt"

type ErrorNotFound struct {
	Entity string
	ID     string
}

func (e *ErrorNotFound) Error() string {
	return fmt.Sprintf("%s with ID '%s' not found", e.Entity, e.ID)
}

type ErrorBadRequest struct {
	Message string
	Field   string
}

func (e *ErrorBadRequest) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("bad request: invalid field '%s' - %s", e.Field, e.Message)
	}
	return fmt.Sprintf("bad request: %s", e.Message)
}

type ErrorConflict struct {
	Message string
}

func (e *ErrorConflict) Error() string {
	return fmt.Sprintf("conflict: %s", e.Message)
}
