package auth_handlers

import (
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/auth"
	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	authErrors = map[string]httperror.HttpError{
		auth.ErrNameIsEmpty.Error():       httperror.NewExternalHttpError(auth.ErrNameIsEmpty, http.StatusBadRequest),
		auth.ErrUserNotExist.Error():      httperror.NewExternalHttpError(auth.ErrUserNotExist, http.StatusBadRequest),
		auth.ErrWrongPassword.Error():     httperror.NewExternalHttpError(auth.ErrWrongPassword, http.StatusBadRequest),
		auth.ErrUserAlreadyExists.Error(): httperror.NewExternalHttpError(auth.ErrUserAlreadyExists, http.StatusContinue),
	}
)

/*
args values:

	0 - function name (for internal errors)
*/
func writeError(w http.ResponseWriter, err error, args ...string) {
	if errors.Is(errors.Unwrap(err), auth.ErrInternal) {
		httperror.NewInternalHttpError(err, args[0]).Write(w)
	} else {
		authErrors[err.Error()].Write(w)
	}
}
