package data_handlers

import (
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	ErrFailedReadBody   = httperror.NewInternalHttpError(errors.New("failed read request body"), "") // Use WithDesc() and WithFuncName() to write response
	ErrRequestBodyEmpty = httperror.NewExternalHttpError(errors.New("request body empty"), http.StatusBadRequest)
	ErrBadJsonBody      = httperror.NewExternalHttpError(errors.New("bad request json body"), http.StatusBadRequest)
)
