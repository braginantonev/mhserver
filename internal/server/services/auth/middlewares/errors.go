package auth_middlewares

import (
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	ErrGetJWTClaims httperror.HttpError = httperror.NewInternalHttpError(errors.New("failed get jwt claims"), "WithAuth")

	ErrUserNotAuthorized   httperror.HttpError = httperror.NewExternalHttpError(errors.New("user not authorized"), http.StatusUnauthorized)
	ErrBadJWTToken         httperror.HttpError = httperror.NewExternalHttpError(errors.New("bad jwt token"), http.StatusBadRequest)
	ErrJwtSignatureInvalid httperror.HttpError = httperror.NewExternalHttpError(errors.New("jwt signature is invalid"), http.StatusBadRequest)

	ErrAuthorizationExpired httperror.HttpError = httperror.NewExternalHttpError(errors.New("authorization expired"), http.StatusUnauthorized)
)
