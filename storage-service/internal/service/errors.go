package service

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrForbidden       = errors.New("forbidden")
	ErrQuotaExceeded   = errors.New("storage quota exceeded")
	ErrFileTooLarge    = errors.New("file too large")
	ErrMimeNotAllowed  = errors.New("file type not allowed")
	ErrInvalidFileName = errors.New("invalid file name")
	ErrShareExpired    = errors.New("share link expired")
)
