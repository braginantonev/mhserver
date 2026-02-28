package datahandler

import (
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/internal/grpc/data"
	"github.com/braginantonev/mhserver/pkg/httperror"
	"google.golang.org/grpc/status"
)

var (
	// Service errors
	ErrInternal          = httperror.NewInternalHttpError(errors.New("internal error"), "")
	ErrUnavailable       = httperror.NewExternalHttpError(errors.New("service is off or unavailable"), http.StatusServiceUnavailable)
	SpecialServiceErrors = map[string]httperror.HttpError{
		data.ErrNotEnoughDiskSpace.Error(): httperror.NewExternalHttpError(data.ErrNotEnoughDiskSpace, http.StatusRequestEntityTooLarge),
	}

	// Handler errors
	ErrWrongContextUsername = httperror.NewInternalHttpError(errors.New("context username from jwt is not string"), "")
	ErrBadQuery             = httperror.NewExternalHttpError(errors.New("bad query format"), http.StatusBadRequest)

	// Data info errors
	ErrNullFileSize = httperror.NewExternalHttpError(errors.New("file size is null"), http.StatusBadRequest)
)

func handleServiceError(err error, w http.ResponseWriter, func_name string) {
	st, ok := status.FromError(err)
	if !ok {
		return
	}

	mess := st.Message()

	if mess == data.ErrInternal.Error() {
		ErrInternal.WithFuncName(func_name).Write(w)
		return
	}

	herr, ok := SpecialServiceErrors[mess]
	if ok {
		herr.Write(w)
		return
	}

	httperror.NewExternalHttpError(errors.New(mess), http.StatusBadRequest).Write(w)
}
