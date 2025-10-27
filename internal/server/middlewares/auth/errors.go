package auth

import (
	"errors"
	"net/http"

	htypes "github.com/braginantonev/mhserver/pkg/handler_types"
)

var (
	HErrGetJWTClaims htypes.HandlerError = htypes.NewInternalHandlerError(errors.New("failed get jwt claims"), "WithAuth")

	HErrUserNotRegistered htypes.HandlerError = htypes.NewExternalHandlerError(errors.New("user not registered"), http.StatusUnauthorized)
	HErrUserNotAuthorized htypes.HandlerError = htypes.NewExternalHandlerError(errors.New("user not authorized"), http.StatusUnauthorized)
	HErrBadJWTToken       htypes.HandlerError = htypes.NewExternalHandlerError(errors.New("bad jwt token"), http.StatusBadRequest)
	HErrBadJWTSignature   htypes.HandlerError = htypes.NewExternalHandlerError(errors.New("bad jwt token signature"), http.StatusBadRequest)
)
