package domain

import (
	"time"

	"github.com/google/uuid"
)

// ProductType is the top-level storefront navigation tier (e.g. Jewellery,
// Clothing) — categories belong to exactly one type, and types drive the
// main nav menu, with each type's categories shown as its dropdown.
type ProductType struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
}
