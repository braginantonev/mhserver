package data

import (
	"errors"
)

var (
	// File find
	ErrBadUUID              error = errors.New("bad file uuid")
	ErrUnexpectedFileChange error = errors.New("unexpected file change")

	// Chunks
	ErrIncorrectChunkSize error = errors.New("incorrect chunk size")

	ErrEmptyFilename      error = errors.New("file name is empty")
	ErrWrongAction        error = errors.New("wrong action")
	ErrUnexpectedFileType error = errors.New("unexpected file type")
	ErrFileNotExist       error = errors.New("file not exist")
	ErrInternal           error = errors.New("internal error")
)
