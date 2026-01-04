package fileuuidmap

import "errors"

var (
	ErrFileNotFound = errors.New("file not found. Bad uuid")
	EOC             = errors.New("end of connection")
)
