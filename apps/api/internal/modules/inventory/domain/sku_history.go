package domain

import (
	"time"

	"github.com/google/uuid"
)

// SKUChange is one entry in an inventory item's SKU audit trail — recorded
// whenever an admin changes the item's SKU. Append-only; never mutated.
type SKUChange struct {
	ID              uuid.UUID
	InventoryItemID uuid.UUID
	OldSKU          string
	NewSKU          string
	// Reason is the optional free-text note the admin gave for the change
	// (e.g. "realigned prefix after moving product to Jewellery").
	Reason    string
	ChangedBy *uuid.UUID
	ChangedAt time.Time
}
