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

	ErrRequestBodyEmpty = httperror.NewExternalHttpError(errors.New("request body empty"), http.StatusBadRequest)
	ErrBadJsonBody      = httperror.NewExternalHttpError(errors.New("bad request json body"), http.StatusBadRequest)
	ErrFailedReadBody      = httperror.NewInternalHttpError(errors.New("failed read request body"), "") // Use WithDesc() and WithFuncName() to write response
)

/*
Wrapper for "github.com/braginantonev/mhserver/pkg/auth" errors. Also that is a shit

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
