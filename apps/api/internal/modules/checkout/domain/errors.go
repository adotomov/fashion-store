package domain

import "errors"

var (
	ErrCartEmpty                 = errors.New("cart is empty")
	ErrInvalidDeliveryMethod     = errors.New("invalid delivery method")
	ErrInvalidPaymentMethod      = errors.New("invalid payment method")
	ErrInsufficientStock         = errors.New("not enough stock available")
	ErrPaymentFailed             = errors.New("payment failed")
	ErrDeliveryMethodUnavailable = errors.New("delivery method is currently unavailable")
	ErrOfficeRequired            = errors.New("a pickup locker is required for this delivery method")
)

type ValidationError string

func (e ValidationError) Error() string { return string(e) }
