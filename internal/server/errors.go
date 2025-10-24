package server

import "errors"

var (
	ErrFailedStartServer error = errors.New("failed to start server")
)
