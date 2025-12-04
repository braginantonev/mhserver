package data_handlers

import (
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	ErrInternal = httperror.NewInternalHttpError(errors.New("internal error"), "")

	ErrWrongContextUsername = httperror.NewInternalHttpError(errors.New("context username from jwt is not string"), "")
	ErrFailedReadBody       = httperror.NewInternalHttpError(errors.New("failed read request body"), "") // Use WithDesc() and WithFuncName() to write response
	ErrRequestBodyEmpty     = httperror.NewExternalHttpError(errors.New("request body empty"), http.StatusBadRequest)
	ErrBadJsonBody          = httperror.NewExternalHttpError(errors.New("bad request json body"), http.StatusBadRequest)

	// Data info errors
	ErrEmptyFilePart = httperror.NewExternalHttpError(errors.New("empty file part"), http.StatusBadRequest)
)
