package authhandler

import (
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/internal/service/auth"
	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	// by default errors have 400 status code
	authSpecialCodes = map[error]int{
		auth.ErrUserAlreadyExists:    http.StatusConflict,
		auth.ErrRegSecretKeyNotFound: http.StatusForbidden,
	}

	// Handler
	ErrInternal          = httperror.NewInternalHttpError("", "")
	ErrRequestBodyEmpty  = httperror.NewExternalHttpError("request body empty", http.StatusBadRequest)
	ErrBadJsonBody       = httperror.NewExternalHttpError("bad request json body", http.StatusBadRequest)
	ErrFailedReadBody    = httperror.NewInternalHttpError("failed read request body", "") // Use WithDesc() and WithFuncName() to write response
	ErrUsernameEmpty     = httperror.NewExternalHttpError("username is empty", http.StatusBadRequest)
	ErrRegSecretKeyEmpty = httperror.NewExternalHttpError("register secret key is empty", http.StatusBadRequest)

	// Middleware
	ErrGetJWTClaims = httperror.NewInternalHttpError("failed get jwt claims", "AuthMiddleware.WithAuth")

	ErrToManyRequests = httperror.NewExternalHttpError("to many requests", http.StatusTooManyRequests)

	ErrUserNotAuthorized   = httperror.NewExternalHttpError("user not authorized", http.StatusUnauthorized)
	ErrBadJWTToken         = httperror.NewExternalHttpError("bad jwt token", http.StatusBadRequest)
	ErrJwtSignatureInvalid = httperror.NewExternalHttpError("jwt signature is invalid", http.StatusBadRequest)

	ErrAuthorizationExpired = httperror.NewExternalHttpError("authorization expired", http.StatusUnauthorized)
)

func handleServiceError(w http.ResponseWriter, err error, func_name string) {
	if errors.Is(err, auth.ErrInternal) {
		ErrInternal.WithFuncName(func_name).Write(w)
	} else {
		cd, ok := authSpecialCodes[err]
		if !ok {
			cd = http.StatusBadRequest
		}
		httperror.NewExternalHttpError(err.Error(), cd).Write(w)
	}
}
