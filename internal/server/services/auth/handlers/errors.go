package auth_handlers

import (
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/auth"
	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	authErrors = map[error]httperror.HttpError{
		auth.ErrNameIsEmpty:       httperror.NewExternalHttpError(auth.ErrNameIsEmpty, http.StatusBadRequest),
		auth.ErrUserNotExist:      httperror.NewExternalHttpError(auth.ErrUserNotExist, http.StatusBadRequest),
		auth.ErrWrongPassword:     httperror.NewExternalHttpError(auth.ErrWrongPassword, http.StatusBadRequest),
		auth.ErrUserAlreadyExists: httperror.NewExternalHttpError(auth.ErrUserAlreadyExists, http.StatusContinue),
	}

	ErrInternal         = httperror.NewInternalHttpError(errors.New(""), "")
	ErrRequestBodyEmpty = httperror.NewExternalHttpError(errors.New("request body empty"), http.StatusBadRequest)
	ErrBadJsonBody      = httperror.NewExternalHttpError(errors.New("bad request json body"), http.StatusBadRequest)
	ErrFailedReadBody   = httperror.NewInternalHttpError(errors.New("failed read request body"), "") // Use WithDesc() and WithFuncName() to write response
)

func handleServiceError(w http.ResponseWriter, err error, func_name string) {
	if errors.Is(errors.Unwrap(err), auth.ErrInternal) {
		ErrInternal.Append(err).WithFuncName(func_name).Write(w)
	} else {
		authErrors[err].Write(w)
	}
}
