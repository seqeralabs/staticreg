package errs

import "errors"

var (
	ErrNotFound         = errors.New("not found")
	ErrInvalidReference = errors.New("invalid reference")
)
