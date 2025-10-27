package handlertypes

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

type HandlerErrorType int

const (
	// Errors messages
	BAD_ERROR string = "error text doesn't match. Won't `%s`, but got `%s`"
	BAD_CODE  string = "error code don't match. Won't `%d`, but got `%d`"
	BAD_TYPE  string = "error type don't match. Won't `%v`, but got `%v`"

	INTERNAL HandlerErrorType = iota
	EXTERNAL
	EMPTY
)

type HandlerError struct {
	error
	Type HandlerErrorType
	Code int
}

// Return nil, if errors not different
func (herr HandlerError) CompareWith(handler_error HandlerError) error {
	if herr.Code != handler_error.Code {
		return fmt.Errorf(BAD_CODE, herr.Code, handler_error.Code)
	}

	if herr.Type != handler_error.Type {
		return fmt.Errorf(BAD_TYPE, herr.Type, handler_error.Type)
	}

	if handler_error.Type != EMPTY && herr.Type != EMPTY && !errors.Is(handler_error, herr) {
		return fmt.Errorf(BAD_ERROR, herr.Error(), handler_error.Error())
	}

	return nil
}

// If error is empty, return true
func (herr HandlerError) Write(w http.ResponseWriter, handler_name string) bool {
	switch herr.Type {
	case INTERNAL:
		slog.Error(herr.Error(), slog.String("handler", handler_name))
		w.WriteHeader(http.StatusInternalServerError)
		return false

	case EXTERNAL:
		http.Error(w, fmt.Sprintf("error: %s", herr.Error()), herr.Code)
		return false
	}

	return true
}

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
