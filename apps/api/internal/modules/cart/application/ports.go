package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/cart/domain"
)

// Repository persists carts and their items. Item mutations return the full,
// freshly-enriched cart so callers (and the HTTP layer) never need a
// separate read round-trip.
type Repository interface {
	FindByID(ctx context.Context, cartID uuid.UUID) (*domain.Cart, error)
	FindByUser(ctx context.Context, userID uuid.UUID) (*domain.Cart, error)
	FindByGuestToken(ctx context.Context, token uuid.UUID) (*domain.Cart, error)
	CreateForUser(ctx context.Context, userID uuid.UUID) (*domain.Cart, error)
	CreateForGuest(ctx context.Context, token uuid.UUID) (*domain.Cart, error)

	AddOrIncrementItem(ctx context.Context, cartID, variantID uuid.UUID, quantity int) (*domain.Cart, error)
	SetItemQuantity(ctx context.Context, cartID, itemID uuid.UUID, quantity int) (*domain.Cart, error)
	RemoveItem(ctx context.Context, cartID, itemID uuid.UUID) (*domain.Cart, error)
	// ClearItems empties a cart (used once an order has been placed from
	// it) without deleting the cart row itself.
	ClearItems(ctx context.Context, cartID uuid.UUID) error

	// MergeCarts moves every item from sourceCartID into targetCartID
	// (summing quantities on conflict), deletes the source cart, and
	// returns the resulting target cart.
	MergeCarts(ctx context.Context, sourceCartID, targetCartID uuid.UUID) (*domain.Cart, error)

	// Checkout-session hold: a single inventory reservation held against the
	// cart for the duration of a checkout, tracked on the cart row.
	SetReservation(ctx context.Context, cartID, reservationID uuid.UUID, expiresAt time.Time) error
	GetReservation(ctx context.Context, cartID uuid.UUID) (*uuid.UUID, *time.Time, error)
	ClearReservation(ctx context.Context, cartID uuid.UUID) error
	ListExpiredReservations(ctx context.Context, cutoff time.Time) ([]domain.ExpiredReservation, error)
}
