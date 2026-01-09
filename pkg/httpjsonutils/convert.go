package httpjsonutils

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	ErrFailedReadBody   = httperror.NewInternalHttpError(errors.New("failed read request body"), "") // Use WithDesc() and WithFuncName() to write response
	ErrRequestBodyEmpty = httperror.NewExternalHttpError(errors.New("request body empty"), http.StatusBadRequest)
	ErrBadJsonBody      = httperror.NewExternalHttpError(errors.New("bad request json body"), http.StatusBadRequest)
)

func ConvertJsonToStruct[T any](s *T, body io.ReadCloser, handler_name string) httperror.HttpError {
	read, err := io.ReadAll(body)
	if err != nil {
		return ErrFailedReadBody.Append(err).WithFuncName(handler_name + ".io.ReadAll")
	}

	if len(read) == 0 {
		return ErrRequestBodyEmpty
	}

	if err = json.Unmarshal(read, s); err != nil {
		return ErrBadJsonBody.Append(err)
	}

	return httperror.HttpError{}
}
