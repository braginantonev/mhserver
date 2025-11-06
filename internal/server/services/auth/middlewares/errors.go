package auth_middlewares

import (
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	HErrGetJWTClaims httperror.HandlerError = httperror.NewInternalHandlerError(errors.New("failed get jwt claims"), "WithAuth")

	HErrUserNotRegistered httperror.HandlerError = httperror.NewExternalHandlerError(errors.New("user not registered"), http.StatusUnauthorized)
	HErrUserNotAuthorized httperror.HandlerError = httperror.NewExternalHandlerError(errors.New("user not authorized"), http.StatusUnauthorized)
	HErrBadJWTToken       httperror.HandlerError = httperror.NewExternalHandlerError(errors.New("bad jwt token"), http.StatusBadRequest)
	HErrBadJWTSignature   httperror.HandlerError = httperror.NewExternalHandlerError(errors.New("bad jwt token signature"), http.StatusBadRequest)
)
