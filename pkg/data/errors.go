package data

import (
	"errors"
	"io"
)

var (
	EOF error = io.EOF

	ErrWrongAction  error = errors.New("wrong action")
	ErrFileNotExist error = errors.New("file not exist")
	ErrInternal     error = errors.New("internal error")
)
