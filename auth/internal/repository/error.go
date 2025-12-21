package repository

import "errors"

// ErrNotFound is returned when the user is not found.
var ErrNotFound = errors.New("not found")
