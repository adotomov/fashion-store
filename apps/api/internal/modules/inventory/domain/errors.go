package domain

import "errors"

var (
	ErrItemNotFound          = errors.New("inventory item not found")
	ErrItemAlreadyExists     = errors.New("inventory item already exists for this variant")
	ErrVariantNotFound       = errors.New("product variant not found")
	ErrSKUConflict           = errors.New("sku already in use")
	ErrInvalidMovementType   = errors.New("invalid or non-admin-adjustable movement type")
	ErrInsufficientStock     = errors.New("adjustment would result in negative stock on hand")
	ErrReservationNotFound   = errors.New("reservation not found")
	ErrReservationNotPending = errors.New("reservation is not pending")
)

type ValidationError string

func (e ValidationError) Error() string { return string(e) }
