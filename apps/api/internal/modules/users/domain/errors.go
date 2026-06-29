package domain

import "errors"

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrAddressNotFound = errors.New("address not found")
)

type ValidationError string

func (e ValidationError) Error() string { return string(e) }

func ErrAddressInvalid(msg string) error { return ValidationError(msg) }
