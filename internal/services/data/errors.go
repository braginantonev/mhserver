package data

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// File find
	ErrBadUUID              error = errors.New("bad connection uuid")           // deprecated
	ErrConnectionNotFound   error = errors.New("connection not found or ended") // deprecated
	ErrUnexpectedFileChange error = status.Error(codes.OutOfRange, "unexpected file change")

	// Chunks
	ErrIncorrectChunkSize error = status.Error(codes.InvalidArgument, "incorrect chunk size")

	// Directory errors
	ErrDirNotFound     error = status.Error(codes.NotFound, "directory not found")
	ErrDirAlreadyExist error = status.Error(codes.AlreadyExists, "directory already exist")

	// Filename errors
	ErrEmptyFilename     error = status.Error(codes.InvalidArgument, "file name is empty")
	ErrBadFilenameSyntax error = status.Error(codes.InvalidArgument, "filename have bad syntax")

	// Connection errors
	ErrNullSizeToSave     error = status.Error(codes.InvalidArgument, "null size to save")
	ErrNotEnoughDiskSpace error = status.Error(codes.ResourceExhausted, "not enough disk space")

	// GetData errors
	ErrFileNotExist  error = status.Error(codes.NotFound, "file not exist")
	ErrReadOutOfFile error = status.Error(codes.OutOfRange, "reading outside of file")

	ErrInternal error = status.Error(codes.Internal, "internal error")
)
