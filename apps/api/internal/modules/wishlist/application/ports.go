package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/wishlist/domain"
)

// Repository persists which products a user has wishlisted, returning items
// enriched with current catalog data so callers never need a separate read
// round-trip.
type Repository interface {
	List(ctx context.Context, userID uuid.UUID) ([]domain.Item, error)
	// Add is idempotent: wishlisting an already-wishlisted product just
	// returns the existing item.
	Add(ctx context.Context, userID, productID uuid.UUID) (*domain.Item, error)
	Remove(ctx context.Context, userID, productID uuid.UUID) error
}
