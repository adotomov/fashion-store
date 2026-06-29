package domain

import "errors"

var ErrPaymentMethodNotFound = errors.New("payment method not found")

type ValidationError string

func (e ValidationError) Error() string { return string(e) }
