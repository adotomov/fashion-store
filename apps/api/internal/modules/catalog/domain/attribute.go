package domain

import (
	"time"

	"github.com/google/uuid"
)

// AttributeType distinguishes plain-text attributes (Size, Material) from
// color attributes, whose values carry a palette color rendered as a swatch.
type AttributeType string

const (
	AttributeTypeText  AttributeType = "text"
	AttributeTypeColor AttributeType = "color"
)

type Attribute struct {
	ID   uuid.UUID
	Name string
	Type AttributeType
	// IsSystem marks a built-in attribute (the "Default" set, e.g. Color)
	// that ships with the store and can't be deleted by admins.
	IsSystem  bool
	Values    []AttributeValue
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AttributeValue struct {
	ID          uuid.UUID
	AttributeID uuid.UUID
	Value       string
	// ColorHex is the picked palette color for color-typed attributes
	// (e.g. "#B2543C"), nil for plain-text attributes.
	ColorHex  *string
	CreatedAt time.Time
}

// AttributeFacet is a storefront-facing view of an attribute and the subset
// of its values actually in use by active products' variants — distinct
// from Attribute, which lists every admin-defined value regardless of use.
type AttributeFacet struct {
	AttributeID   uuid.UUID
	AttributeName string
	AttributeType AttributeType
	Values        []AttributeFacetValue
}

type AttributeFacetValue struct {
	ID       uuid.UUID
	Value    string
	ColorHex *string
}
