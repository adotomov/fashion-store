package domain

import (
	"time"

	"github.com/google/uuid"
)

type InventoryItem struct {
	ID               uuid.UUID
	VariantID        uuid.UUID
	SKU              string
	QuantityOnHand   int
	QuantityReserved int
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// Display-only fields populated by list/get queries that join through
	// to the product/variant — never set when constructing an item to
	// create. Catalog and inventory stay separate at the persistence layer;
	// this is purely a read convenience for the admin UI.
	ProductName  string
	VariantLabel string
}

func (i InventoryItem) QuantityAvailable() int {
	return i.QuantityOnHand - i.QuantityReserved
}
