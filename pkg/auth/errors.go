package auth

import "errors"

var (
	// External errors
	ErrNameIsEmpty error = errors.New("username is empty")

	// - Login errors
	ErrUserNotExist  error = errors.New("wrong username or user not registered")
	ErrWrongPassword error = errors.New("wrong password")

	// - Register errors
	ErrUserAlreadyExists error = errors.New("user already registered")
)
