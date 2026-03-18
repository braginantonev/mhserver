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

	// by default errors have 400 status code
	SpecialCodes = map[string]int{
		data.ErrNotEnoughDiskSpace.Error():   http.StatusRequestEntityTooLarge,
		data.ErrUnexpectedFileChange.Error(): http.StatusForbidden,
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

	cd, ok := SpecialCodes[mess]
	if !ok {
		cd = http.StatusBadRequest
	}

	httperror.NewExternalHttpError(mess, cd).Write(w)
}
