package auth_middlewares

import (
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	HErrGetJWTClaims httperror.HttpError = httperror.NewInternalHttpError(errors.New("failed get jwt claims"), "WithAuth")

	HErrUserNotRegistered httperror.HttpError = httperror.NewExternalHttpError(errors.New("user not registered"), http.StatusUnauthorized)
	HErrUserNotAuthorized httperror.HttpError = httperror.NewExternalHttpError(errors.New("user not authorized"), http.StatusUnauthorized)
	HErrBadJWTToken       httperror.HttpError = httperror.NewExternalHttpError(errors.New("bad jwt token"), http.StatusBadRequest)
	HErrBadJWTSignature   httperror.HttpError = httperror.NewExternalHttpError(errors.New("bad jwt token signature"), http.StatusBadRequest)
)
