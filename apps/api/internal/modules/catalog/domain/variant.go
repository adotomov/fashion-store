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
	// Physical shipping attributes, set per SKU. Used to compute a parcel's
	// weight when asking Speedy for shipping costs. Zero means "not set".
	// Weight is in grams; dimensions in centimetres.
	WeightGrams int
	LengthCM    int
	WidthCM     int
	HeightCM    int
	Attributes  []AttributeValue
	// InventoryItemID and QuantityAvailable are read-only enrichment from the
	// inventory module (joined at the SQL level, same as the cart module
	// does) — nil/nil means no inventory item has been assigned to this
	// variant yet, which the storefront treats as out of stock.
	InventoryItemID   *uuid.UUID
	QuantityAvailable *int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
