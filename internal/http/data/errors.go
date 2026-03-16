package datahandler

import (
	"net/http"

	"github.com/braginantonev/mhserver/internal/grpc/data"
	"github.com/braginantonev/mhserver/pkg/httperror"
	"google.golang.org/grpc/status"
)

var (
	// Service errors
	ErrInternal    = httperror.NewInternalHttpError("", "")
	ErrUnavailable = httperror.NewExternalHttpError("service is off or unavailable", http.StatusServiceUnavailable)

	// * I set empty desc for this errors, because they should be added in the error handler
	SpecialServiceErrors = map[string]httperror.HttpError{
		data.ErrNotEnoughDiskSpace.Error():   httperror.NewExternalHttpError("", http.StatusRequestEntityTooLarge),
		data.ErrUnexpectedFileChange.Error(): httperror.NewExternalHttpError("", http.StatusForbidden),
	}

	// Handler errors
	ErrWrongContextUsername = httperror.NewInternalHttpError("context username from jwt is not string", "")
	ErrBadUuidFormat        = httperror.NewExternalHttpError("bad uuid format", http.StatusBadRequest)

	// Data info errors
	ErrNullFileSize = httperror.NewExternalHttpError("file size is null", http.StatusBadRequest)
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
		herr.AppendStr(mess).Write(w)
		return
	}

	httperror.NewExternalHttpError(mess, http.StatusBadRequest).Write(w)
}
