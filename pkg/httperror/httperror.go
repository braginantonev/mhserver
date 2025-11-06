package httperror

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

type HttpErrorType int

const (
	// Errors messages
	BAD_ERROR string = "error text doesn't match. Won't `%s`, but got `%s`"
	BAD_CODE  string = "error code don't match. Won't `%d`, but got `%d`"
	BAD_TYPE  string = "error type don't match. Won't `%v`, but got `%v`"

	INTERNAL HttpErrorType = iota
	EXTERNAL
	EMPTY
)

type HttpError struct {
	error
	Type HttpErrorType
	Code int

	funcName string // for internal errors only
}

// Return nil, if errors not different
func (herr HttpError) CompareWith(http_error HttpError) error {
	if herr.Code != http_error.Code {
		return fmt.Errorf(BAD_CODE, herr.Code, http_error.Code)
	}

	if herr.Type != http_error.Type {
		return fmt.Errorf(BAD_TYPE, herr.Type, http_error.Type)
	}

	if http_error.Type != EMPTY && herr.Type != EMPTY && !errors.Is(http_error, herr) {
		return fmt.Errorf(BAD_ERROR, herr.Error(), http_error.Error())
	}

	return nil
}

// If error is empty, return true
func (herr HttpError) Write(w http.ResponseWriter) bool {
	switch herr.Type {
	case INTERNAL:
		slog.Error(herr.Error(), slog.String("handler", herr.funcName))
		w.WriteHeader(http.StatusInternalServerError)
		return false

	case EXTERNAL:
		http.Error(w, fmt.Sprintf("error: %s", herr.Error()), herr.Code)
		return false
	}

	return true
}

func NewInternalHttpError(err error, func_name string) HttpError {
	return HttpError{
		error:    err,
		Type:     INTERNAL,
		Code:     http.StatusInternalServerError,
		funcName: func_name,
	}
}

func NewExternalHttpError(err error, http_code int) HttpError {
	return HttpError{
		error: err,
		Type:  EXTERNAL,
		Code:  http_code,
	}
}

func NewEmptyHttpError() HttpError {
	return HttpError{
		Type: EMPTY,
	}
}
