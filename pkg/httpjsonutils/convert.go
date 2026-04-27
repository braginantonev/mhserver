package httpjsonutils

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/httperror"
)

var (
	ErrRequestBodyEmpty = httperror.NewExternalHttpError("request body empty", http.StatusBadRequest)
	ErrBadJsonBody      = httperror.NewExternalHttpError("bad request json body", http.StatusBadRequest)
)

func ConvertJsonToStruct[T any](s *T, body io.ReadCloser, handler_name string) httperror.HttpError {
	if err := json.NewDecoder(body).Decode(s); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrRequestBodyEmpty
		}
		return ErrBadJsonBody.Append(err)
	}
	return nil
}
