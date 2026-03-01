package authhandler

import (
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/internal/service/auth"
	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	// * I set empty desc for this errors, because they should be added in the error handler
	authErrors = map[error]httperror.HttpError{
		auth.ErrNameIsEmpty:       httperror.NewExternalHttpError("", http.StatusBadRequest),
		auth.ErrUserNotExist:      httperror.NewExternalHttpError("", http.StatusBadRequest),
		auth.ErrWrongPassword:     httperror.NewExternalHttpError("", http.StatusBadRequest),
		auth.ErrUserAlreadyExists: httperror.NewExternalHttpError("", http.StatusConflict),
	}

	// Handler
	ErrInternal         = httperror.NewInternalHttpError("", "")
	ErrRequestBodyEmpty = httperror.NewExternalHttpError("request body empty", http.StatusBadRequest)
	ErrBadJsonBody      = httperror.NewExternalHttpError("bad request json body", http.StatusBadRequest)
	ErrFailedReadBody   = httperror.NewInternalHttpError("failed read request body", "") // Use WithDesc() and WithFuncName() to write response

	// Middleware
	ErrGetJWTClaims = httperror.NewInternalHttpError("failed get jwt claims", "AuthMiddleware.WithAuth")

	ErrUserNotAuthorized   = httperror.NewExternalHttpError("user not authorized", http.StatusUnauthorized)
	ErrBadJWTToken         = httperror.NewExternalHttpError("bad jwt token", http.StatusBadRequest)
	ErrJwtSignatureInvalid = httperror.NewExternalHttpError("jwt signature is invalid", http.StatusBadRequest)

	ErrAuthorizationExpired = httperror.NewExternalHttpError("authorization expired", http.StatusUnauthorized)
)

func handleServiceError(w http.ResponseWriter, err error, func_name string) {
	if errors.Is(err, auth.ErrInternal) {
		ErrInternal.WithFuncName(func_name).Write(w)
	} else {
		authErrors[err].Append(err).Write(w)
	}
}
