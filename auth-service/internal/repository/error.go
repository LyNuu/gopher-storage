package repository

import "errors"

var (
	ErrEmailAlreadyExists = errors.New("email already exist")
	ErrUserNotFound       = errors.New("user not found")
)
