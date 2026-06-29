package domain

import (
	"time"

	"github.com/google/uuid"
)

type Attribute struct {
	ID        uuid.UUID
	Name      string
	Values    []AttributeValue
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AttributeValue struct {
	ID          uuid.UUID
	AttributeID uuid.UUID
	Value       string
	CreatedAt   time.Time
}

// AttributeFacet is a storefront-facing view of an attribute and the subset
// of its values actually in use by active products' variants — distinct
// from Attribute, which lists every admin-defined value regardless of use.
type AttributeFacet struct {
	AttributeID   uuid.UUID
	AttributeName string
	Values        []AttributeFacetValue
}

type AttributeFacetValue struct {
	ID    uuid.UUID
	Value string
}
