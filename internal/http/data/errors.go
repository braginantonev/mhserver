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

	// Data info errors
	ErrNullFileSize = httperror.NewExternalHttpError(errors.New("file size is null"), http.StatusBadRequest)
)

func handleServiceError(err error, w http.ResponseWriter, func_name string) {
	if errors.Is(errors.Unwrap(err), data.ErrInternal) {
		ErrInternal.Append(err).WithFuncName(func_name).Write(w)
	} else {
		httperror.NewExternalHttpError(err, http.StatusBadRequest).Write(w)
	}
}
