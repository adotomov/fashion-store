package domain

import "errors"

var (
	ErrCartEmpty                 = errors.New("cart is empty")
	ErrInvalidDeliveryMethod     = errors.New("invalid delivery method")
	ErrInvalidPaymentMethod      = errors.New("invalid payment method")
	ErrPaymentMethodNotAllowed   = errors.New("payment method is not available for the chosen delivery method")
	ErrInsufficientStock         = errors.New("not enough stock available")
	ErrPaymentFailed             = errors.New("payment failed")
	ErrDeliveryMethodUnavailable = errors.New("delivery method is currently unavailable")
	ErrOfficeRequired            = errors.New("a pickup locker is required for this delivery method")
	ErrInvalidDiscountCode       = errors.New("discount code is invalid, expired, or exhausted")
)

type ValidationError string

func (e ValidationError) Error() string { return string(e) }
