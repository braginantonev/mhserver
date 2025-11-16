package services

import (
	"errors"
)

var (
	ErrDBNotInit           error = errors.New("db not initialized")
	ErrJWTSigIsEmpty       error = errors.New("jwt signature is empty")
	ErrFailedWriteResponse error = errors.New("failed write response")
)
