package data

import (
	"errors"
)

var (
	// File find
	ErrBadUUID              error = errors.New("bad connection uuid")
	ErrUnexpectedFileChange error = errors.New("unexpected file change")

	// Chunks
	ErrIncorrectChunkSize error = errors.New("incorrect chunk size")

	// CreateConnection errors
	ErrEmptyFilename      error = errors.New("file name is empty")
	ErrUnexpectedFileType error = errors.New("unexpected file type")
	ErrNotEnoughDiskSpace error = errors.New("not enough disk space")

	// GetData errors
	ErrFileNotExist error = errors.New("file not exist")

	// Directory errors
	ErrDirNotFound     error = errors.New("directory not found")
	ErrEmptyDir        error = errors.New("directory is empty")
	ErrDirAlreadyExist error = errors.New("directory already exist")
	ErrBadDirSyntax    error = errors.New("directory have bad syntax")

	ErrInternal error = errors.New("internal error")
)
