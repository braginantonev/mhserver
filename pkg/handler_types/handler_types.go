package handlertypes

import (
	"errors"
	"net/http"
)

type HandlerErrorType int

const (
	INTERNAL HandlerErrorType = iota
	EXTERNAL
	EMPTY
)

type (
	HandlerError struct {
		Type HandlerErrorType
		Code int
		error
	}
)

func NewInternalHandlerError() HandlerError {
	return HandlerError{
		error: errors.New(""),
		Type:  INTERNAL,
		Code:  http.StatusInternalServerError,
	}
}

func NewExternalHandlerError(err error, http_code int) HandlerError {
	return HandlerError{
		error: err,
		Type:  EXTERNAL,
		Code:  http_code,
	}
}

func NewEmptyHandlerError() HandlerError {
	return HandlerError{
		Type: EMPTY,
	}
}
