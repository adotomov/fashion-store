package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

var (
	ErrItemNotFound    = errors.New("wishlist item not found")
	ErrProductNotFound = errors.New("product not found")
)

// Item is read enriched with current catalog data (name, slug, image,
// price, stock, sizes) rather than snapshotting at add-time — a wishlist
// entry should always reflect what the customer would see on the product
// page right now, same rationale as cart.CartItem.
type Item struct {
	ID             uuid.UUID
	ProductID      uuid.UUID
	ProductName    string
	ProductSlug    string
	ImageMediaID   *uuid.UUID
	BasePrice      money.Money
	CompareAtPrice *money.Money
	InStock        bool
	// Sizes lists the distinct "Size" attribute values across the
	// product's variants, in no particular order — used to preview what's
	// available without visiting the product page.
	Sizes     []string
	CreatedAt time.Time
}
