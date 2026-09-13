package auth

import (
	"errors"
)

var (
	ErrInternal error = errors.New("internal error")

	//JWT errors
	ErrJwtSignatureInvalid error = errors.New("wrong token signature")
	ErrWrongJWTName        error = errors.New("wrong username from jwt token")

	// External errors
	ErrEmptyRequest         error = errors.New("request is empty")
	ErrNameTooLong          error = errors.New("name is too long")
	ErrNullUsername         error = errors.New("username is null")
	ErrRegSecretKeyNotFound error = errors.New("wrong register secret key")

	// - Login errors
	ErrUserNotExist  error = errors.New("wrong username or user not registered")
	ErrWrongPassword error = errors.New("wrong password")

	// - Register errors
	ErrUserAlreadyExists error = errors.New("user already registered")
)
