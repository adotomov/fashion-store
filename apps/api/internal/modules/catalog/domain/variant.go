package domain

import (
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type ProductVariant struct {
	ID            uuid.UUID
	ProductID     uuid.UUID
	PriceOverride *money.Money
	Attributes    []AttributeValue
	// InventoryItemID and QuantityAvailable are read-only enrichment from the
	// inventory module (joined at the SQL level, same as the cart module
	// does) — nil/nil means no inventory item has been assigned to this
	// variant yet, which the storefront treats as out of stock.
	InventoryItemID   *uuid.UUID
	QuantityAvailable *int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
