package middlewares

import "errors"

var (
	// Internal
	ErrFailedReadRequestBody error = errors.New("failed read request body")
)
