package auth

import (
	"errors"
)

var (
	ErrInternal error = errors.New("internal error")

	//JWT errors
	ErrJwtSignatureInvalid error = errors.New("wrong token signature")
	ErrWrongJWTName        error = errors.New("wrong username from jwt token")
	ErrBadClaims           error = errors.New("failed get claims from jwt token")

	// External errors
	ErrNameIsEmpty error = errors.New("username is empty")

	// - Login errors
	ErrUserNotExist  error = errors.New("wrong username or user not registered")
	ErrWrongPassword error = errors.New("wrong password")

	// - Register errors
	ErrUserAlreadyExists error = errors.New("user already registered")

	
)
