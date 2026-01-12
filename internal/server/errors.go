package server

import "errors"

var (
	ErrUnsafeProtocol    error = errors.New("tls not configured")
	ErrFailedStartServer error = errors.New("failed to start server")
)
