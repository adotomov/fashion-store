package domain

import "errors"

var (
	ErrPromotionNotFound = errors.New("promotion not found")
	ErrCodeNotFound      = errors.New("discount code not found")
	ErrCodeInvalid       = errors.New("discount code is invalid or expired")
	ErrCodeExhausted     = errors.New("discount code has reached its usage limit")
	ErrDuplicateCode     = errors.New("a discount code with that value already exists")
)
