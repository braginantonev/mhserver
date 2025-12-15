package authhandler

import (
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/internal/service/auth"
	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	authErrors = map[error]httperror.HttpError{
		auth.ErrNameIsEmpty:       httperror.NewExternalHttpError(auth.ErrNameIsEmpty, http.StatusBadRequest),
		auth.ErrUserNotExist:      httperror.NewExternalHttpError(auth.ErrUserNotExist, http.StatusBadRequest),
		auth.ErrWrongPassword:     httperror.NewExternalHttpError(auth.ErrWrongPassword, http.StatusBadRequest),
		auth.ErrUserAlreadyExists: httperror.NewExternalHttpError(auth.ErrUserAlreadyExists, http.StatusContinue),
	}

	// Handler
	ErrInternal         = httperror.NewInternalHttpError(errors.New(""), "")
	ErrRequestBodyEmpty = httperror.NewExternalHttpError(errors.New("request body empty"), http.StatusBadRequest)
	ErrBadJsonBody      = httperror.NewExternalHttpError(errors.New("bad request json body"), http.StatusBadRequest)
	ErrFailedReadBody   = httperror.NewInternalHttpError(errors.New("failed read request body"), "") // Use WithDesc() and WithFuncName() to write response

	// Middleware
	ErrGetJWTClaims = httperror.NewInternalHttpError(errors.New("failed get jwt claims"), "AuthMiddleware.WithAuth")

	ErrUserNotAuthorized   = httperror.NewExternalHttpError(errors.New("user not authorized"), http.StatusUnauthorized)
	ErrBadJWTToken         = httperror.NewExternalHttpError(errors.New("bad jwt token"), http.StatusBadRequest)
	ErrJwtSignatureInvalid = httperror.NewExternalHttpError(errors.New("jwt signature is invalid"), http.StatusBadRequest)

	ErrAuthorizationExpired = httperror.NewExternalHttpError(errors.New("authorization expired"), http.StatusUnauthorized)
)

func handleServiceError(w http.ResponseWriter, err error, func_name string) {
	if errors.Is(errors.Unwrap(err), auth.ErrInternal) {
		ErrInternal.Append(err).WithFuncName(func_name).Write(w)
	} else {
		authErrors[err].Write(w)
	}
}
