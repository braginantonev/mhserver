package data

import (
	"errors"
)

var (
	// File find
	ErrBadUUID              error = errors.New("bad connection uuid")
	ErrConnectionNotFound   error = errors.New("connection not found or ended")
	ErrUnexpectedFileChange error = errors.New("unexpected file change")

	// Chunks
	ErrIncorrectChunkSize error = errors.New("incorrect chunk size")

	// Directory errors
	ErrDirNotFound     error = errors.New("directory not found")
	ErrUnspecifiedDir  error = errors.New("target directory is not specified")
	ErrDirAlreadyExist error = errors.New("directory already exist")
	ErrBadDirSyntax    error = errors.New("directory have bad syntax")

	// Filename errors
	ErrEmptyFilename     error = errors.New("file name is empty")
	ErrBadFilenameSyntax error = errors.New("filename have bad syntax")

	// Connection errors
	ErrUnexpectedFileType error = errors.New("unexpected file type")
	ErrNullSizeToSave     error = errors.New("null size to save")
	ErrNotEnoughDiskSpace error = errors.New("not enough disk space")

	// GetData errors
	ErrFileNotExist  error = errors.New("file not exist")
	ErrReadOutOfFile error = errors.New("reading outside of file")

	ErrInternal error = errors.New("internal error")
)
