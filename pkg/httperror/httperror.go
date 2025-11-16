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
	Type       HttpErrorType
	StatusCode int

	description string
	funcName    string // for internal errors only
}

// Return nil, if errors not different
func (herr HttpError) CompareWith(http_error HttpError) error {
	if herr.Type != http_error.Type {
		return fmt.Errorf(BAD_TYPE, herr.Type, http_error.Type)
	}

	if herr.StatusCode != http_error.StatusCode {
		return fmt.Errorf(BAD_CODE, herr.StatusCode, http_error.StatusCode)
	}

	if http_error.Type != EMPTY && herr.Type != EMPTY && !errors.Is(http_error, herr) {
		return fmt.Errorf(BAD_ERROR, herr.description, http_error.description)
	}

	return nil
}

// If error is empty, return true
func (herr HttpError) Write(w http.ResponseWriter) bool {
	switch herr.Type {
	case INTERNAL:
		slog.Error(herr.description, slog.String("handler", herr.funcName))
		w.WriteHeader(http.StatusInternalServerError)
		return false

	case EXTERNAL:
		w.WriteHeader(herr.StatusCode)
		w.Write([]byte(herr.description))
		return false
	}

	return true
}

func (herr HttpError) Error() string {
	return herr.description
}

func NewInternalHttpError(err error, func_name string) HttpError {
	return HttpError{
		Type:        INTERNAL,
		StatusCode:  http.StatusInternalServerError,
		description: err.Error(),
		funcName:    func_name,
	}
}

func NewExternalHttpError(err error, status_code int) HttpError {
	return HttpError{
		Type:        EXTERNAL,
		StatusCode:  status_code,
		description: err.Error(),
	}
}

func NewEmptyHttpError() HttpError {
	return HttpError{
		Type: EMPTY,
	}
}
