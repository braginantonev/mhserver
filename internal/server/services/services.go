package services

import (
	"log/slog"
	"net/http"
)

func WriteResponse(w http.ResponseWriter, data []byte, code int) {
	w.WriteHeader(code)
	_, err := w.Write(data)
	if err != nil {
		slog.Error(ErrFailedWriteResponse.Error(), slog.String("error", err.Error()))
	}
}
