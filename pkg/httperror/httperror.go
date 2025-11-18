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

	if !errors.Is(http_error, herr) {
		return fmt.Errorf(BAD_ERROR, herr.description, http_error.description)
	}

	return nil
}

func (herr HttpError) Write(w http.ResponseWriter) {
	switch herr.Type {
	case INTERNAL:
		slog.Error(herr.description, slog.String("handler", herr.funcName))
		w.WriteHeader(http.StatusInternalServerError)

	case EXTERNAL:
		w.WriteHeader(herr.StatusCode)

		_, err := w.Write([]byte(herr.description))
		if err != nil {
			slog.Error("error write response", slog.String("error", err.Error()))
		}
	}
}

// Change func name, which will be logged
func (herr HttpError) WithFuncName(func_name string) HttpError {
	herr.funcName = func_name
	return herr
}

/*
Add new info to error description.
For example:

	error "internal error" + error "fatal error" = error "internal error;\nfatal error"
*/
func (herr HttpError) Append(new_err error) HttpError {
	if herr.description == "" {
		herr.description = new_err.Error()
	} else {
		herr.description += ";\n" + new_err.Error()
	}

	return herr
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
