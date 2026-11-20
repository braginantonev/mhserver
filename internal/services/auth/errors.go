package auth

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInternal            error = status.Error(codes.Internal, "internal error")
	ErrJwtSignatureInvalid error = errors.New("wrong token signature")

	// External errors
	ErrEmptyRequest         error = status.Error(codes.InvalidArgument, "request is empty")
	ErrNameTooLong          error = status.Error(codes.InvalidArgument, "name is too long")
	ErrNullUsername         error = status.Error(codes.InvalidArgument, "username is null")
	ErrRegSecretKeyNotFound error = status.Error(codes.PermissionDenied, "register key not found or has been used")

	// - Login errors
	ErrUserNotExist  error = status.Error(codes.NotFound, "wrong username or user not registered")
	ErrWrongPassword error = status.Error(codes.InvalidArgument, "wrong password")

	// - Register errors
	ErrUserAlreadyExists error = status.Error(codes.AlreadyExists, "user already registered")
)
