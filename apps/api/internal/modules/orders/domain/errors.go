package domain

import "errors"

var (
	ErrOrderNotFound       = errors.New("order not found")
	ErrReservationNotFound = errors.New("reservation not found")
)

type ValidationError string

func (e ValidationError) Error() string { return string(e) }
