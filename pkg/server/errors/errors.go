package errors

import "errors"

type ServerError struct {
	error
	UserMessage string
}

func New(err error, userMessage string) ServerError {
	return ServerError{
		error:       err,
		UserMessage: userMessage,
	}
}

var ErrRepositoryNotFound = New(errors.New("repository not found"), "repository not found")
