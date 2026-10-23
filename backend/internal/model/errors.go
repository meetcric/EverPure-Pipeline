package model

import "errors"

var ErrNotFound = errors.New("not found")

var ErrIdempotencyMismatch = errors.New("idempotency key was already used with a different request body")

// ValidationError means the input broke a rule. Handlers map it to 400.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// ErrInvalidTransition means the run's current status does not allow the requested one. Handlers map it to 409.
var ErrInvalidTransition = errors.New("invalid status transition")
