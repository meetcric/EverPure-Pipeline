package model

import "errors"

// ErrNotFound means the requested row does not exist. Handlers map it to 404.
var ErrNotFound = errors.New("not found")

// ValidationError means the input broke a rule. Handlers map it to 400.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }
