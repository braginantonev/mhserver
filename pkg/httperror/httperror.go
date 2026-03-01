package httperror

import (
	"fmt"
	"log/slog"
	"net/http"
)

// This is alias for Error struct
type HttpError = *Error

type Error struct {
	status_code int
	description string
	funcName    string // for internal errors only
}

func (err HttpError) Write(w http.ResponseWriter) {
	if err == nil {
		slog.Error("write to Writer a nil http error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(err.status_code)

	if err.status_code == http.StatusInternalServerError {
		slog.Error("INTERNAL ERROR: ", slog.String("func", err.funcName), slog.String("desc", err.description))
	} else {
		_, _ = w.Write([]byte(err.description))
	}
}

// Change func name, which will be logged
func (err HttpError) WithFuncName(func_name string) HttpError {
	err.funcName = func_name
	return err
}

/*
Add new info to error description.
For example:

	error "internal error" + error "fatal error" = error "internal error;\nfatal error"
*/
func (err HttpError) AppendStr(new_err_str string) HttpError {
	if err.description == "" {
		err.description = new_err_str
	} else {
		err.description += ";\n" + new_err_str
	}

	return err
}

/*
Add new info to error description.
For example:

	error "internal error" + error "fatal error" = error "internal error;\nfatal error"
*/
func (err HttpError) Append(new_err error) HttpError {
	return err.AppendStr(new_err.Error())
}

// Return error description. Use Error() instead to return full error (with status code)
func (err HttpError) Description() string {
	if err == nil {
		return ""
	}

	return err.description
}

// Return error description. Use Error() instead to return full error (with status code)
func (err HttpError) Status() int {
	if err == nil {
		return 0
	}

	return err.status_code
}

// * error interface implementation

// Return full error (description + status code)
func (err HttpError) Error() string {
	return fmt.Sprintf("%s (%d)", err.Description(), err.Status())
}

func (err HttpError) Is(target error) bool {
	if target == err {
		return true
	}

	return err.Error() == target.Error()
}

func NewInternalHttpError(err error, func_name string) HttpError {
	return &Error{
		status_code: http.StatusInternalServerError,
		description: err.Error(),
		funcName:    func_name,
	}
}

func NewExternalHttpError(err error, status_code int) HttpError {
	return &Error{
		status_code: status_code,
		description: err.Error(),
	}
}
