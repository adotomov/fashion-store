package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

var (
	ErrCartNotFound      = errors.New("cart not found")
	ErrCartItemNotFound  = errors.New("cart item not found")
	ErrVariantNotFound   = errors.New("variant not found")
	ErrInsufficientStock = errors.New("insufficient stock")
)

type ValidationError string

func (e ValidationError) Error() string { return string(e) }

type Cart struct {
	ID         uuid.UUID
	UserID     *uuid.UUID
	GuestToken *uuid.UUID
	Items      []CartItem
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ExpiredReservation identifies an abandoned checkout hold: the cart owner it
// belongs to (so the caller can clear the hold on the right cart) and the
// inventory reservation to release. Returned by the abandonment sweeper.
type ExpiredReservation struct {
	UserID        *uuid.UUID
	GuestToken    *uuid.UUID
	ReservationID uuid.UUID
}

// CartItem is read enriched with current catalog/inventory data (price,
// stock, product name/image) rather than snapshotting at add-time — unlike
// an order line, a cart line should always reflect what the customer would
// actually pay/get if they checked out right now.
type CartItem struct {
	ID                uuid.UUID
	CartID            uuid.UUID
	VariantID         uuid.UUID
	ProductID         uuid.UUID
	ProductName       string
	ProductSlug       string
	VariantLabel      string
	ImageMediaID      *uuid.UUID
	UnitPrice         money.Money
	Quantity          int
	AvailableQuantity int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
