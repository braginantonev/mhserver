package datahandler

import (
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/internal/grpc/data"
	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	// GRPC errors
	ErrInternal    = httperror.NewInternalHttpError(errors.New("internal error"), "")
	ErrUnavailable = httperror.NewExternalHttpError(errors.New("service is off or unavailable"), http.StatusServiceUnavailable)

	// Handler errors
	ErrWrongContextUsername = httperror.NewInternalHttpError(errors.New("context username from jwt is not string"), "")
	ErrFailedReadBody       = httperror.NewInternalHttpError(errors.New("failed read request body"), "") // Use WithDesc() and WithFuncName() to write response
	ErrRequestBodyEmpty     = httperror.NewExternalHttpError(errors.New("request body empty"), http.StatusBadRequest)
	ErrBadJsonBody          = httperror.NewExternalHttpError(errors.New("bad request json body"), http.StatusBadRequest)
	ErrNullFileSize         = httperror.NewExternalHttpError(errors.New("file size is null"), http.StatusBadRequest)

	// Data info errors
	ErrEmptyFilePart = httperror.NewExternalHttpError(errors.New("empty file part"), http.StatusBadRequest)
)

func handleServiceError(err error, w http.ResponseWriter, func_name string) {
	if errors.Is(errors.Unwrap(err), data.ErrInternal) {
		ErrInternal.Append(err).WithFuncName(func_name).Write(w)
	} else {
		httperror.NewExternalHttpError(err, http.StatusBadRequest).Write(w)
	}
}
