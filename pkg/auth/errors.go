package auth

import "errors"

var (
	ErrNameIsEmpty error = errors.New("username is empty")

	ErrUserAlreadyExists error = errors.New("user already registered")
)
