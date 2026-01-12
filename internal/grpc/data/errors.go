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

	// CreateConnection errors
	ErrEmptyFilename      error = errors.New("file name is empty")
	ErrUnexpectedFileType error = errors.New("unexpected file type")
	ErrNotEnoughDiskSpace error = errors.New("not enough disk space")

	// GetData errors
	ErrFileNotExist error = errors.New("file not exist")

	ErrInternal error = errors.New("internal error")
)
