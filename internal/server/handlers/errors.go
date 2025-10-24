package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	htypes "github.com/braginantonev/mhserver/pkg/handler_types"
)

var (
	ErrDBNotInit     error = errors.New("db not initialized")
	ErrJWTSigIsEmpty error = errors.New("jwt signature is empty")
)

// If error is empty, return true
func LogError(w http.ResponseWriter, herr htypes.HandlerError, handler_name string) bool {
	switch herr.Type {
	case htypes.INTERNAL:
		slog.Error(herr.Error(), slog.String("handler", handler_name))
		w.WriteHeader(http.StatusInternalServerError)
		return false

	case htypes.EXTERNAL:
		http.Error(w, fmt.Sprintf("error: %s", herr.Error()), herr.Code)
		return false
	}

	return true
}
