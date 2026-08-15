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

// MulticolorHex is a sentinel stored in AttributeValue.ColorHex to mark the
// permanent built-in "Multicolor" swatch, which represents multi-colored
// garments (prints, patterns) and renders as a rainbow wheel instead of a solid
// fill. It is intentionally not a valid hex so it can only be seeded (via
// migration), never created or deleted through the admin API.
const MulticolorHex = "multicolor"

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

// IsMulticolor reports whether this value is the permanent built-in Multicolor
// swatch (see MulticolorHex), which the storefront renders as a rainbow wheel
// and the admin API refuses to delete.
func (v AttributeValue) IsMulticolor() bool {
	return v.ColorHex != nil && *v.ColorHex == MulticolorHex
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
