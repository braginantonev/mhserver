package data

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrBrokenSequence error = status.Error(codes.InvalidArgument, "the sequence is broken")
	ErrInternal       error = status.Error(codes.Internal, "internal error")

	// Chunks
	ErrUnexpectedFileChange error = status.Error(codes.OutOfRange, "unexpected file change")
	ErrIncorrectChunkSize   error = status.Error(codes.InvalidArgument, "incorrect chunk size")

	// Directory errors
	ErrDirNotFound     error = status.Error(codes.NotFound, "directory not found")
	ErrDirAlreadyExist error = status.Error(codes.AlreadyExists, "directory already exist")
	ErrBadDirSyntax    error = status.Error(codes.InvalidArgument, "directory have bad syntax")

	// Filename errors
	ErrEmptyFilename     error = status.Error(codes.InvalidArgument, "file name is empty")
	ErrBadFilenameSyntax error = status.Error(codes.InvalidArgument, "filename have bad syntax")

	// Connection errors
	ErrNullSizeToSave     error = status.Error(codes.InvalidArgument, "null size to save")
	ErrNotEnoughDiskSpace error = status.Error(codes.ResourceExhausted, "not enough disk space")

	// GetData errors
	ErrFileNotExist  error = status.Error(codes.NotFound, "file not exist")
	ErrReadOutOfFile error = status.Error(codes.OutOfRange, "reading outside of file")
)
