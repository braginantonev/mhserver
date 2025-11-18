package services

import (
	"errors"
)

var (
	ErrRequestBodyEmpty    error = errors.New("request body empty")
	ErrDBNotInit           error = errors.New("db not initialized")
	ErrJWTSigIsEmpty       error = errors.New("jwt signature is empty")
	ErrFailedWriteResponse error = errors.New("failed write response")
)
