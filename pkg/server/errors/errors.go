package errors

import "errors"

var ErrRepositoryNotFound = errors.New("repository not found")
var ErrSlugTooShort = errors.New("slug too short")
