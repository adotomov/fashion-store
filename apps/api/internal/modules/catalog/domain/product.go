package domain

import (
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type ProductStatus string

const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusArchived ProductStatus = "archived"
)

func (s ProductStatus) Valid() bool {
	switch s {
	case ProductStatusDraft, ProductStatusActive, ProductStatusArchived:
		return true
	default:
		return false
	}
}

// AttributeRef names an attribute that's relevant to a product, without
// committing to any specific value — the product's variants are what
// carry actual values (e.g. product attributes = [Size, Color], variant =
// {Size: "M", Color: "Blue"}).
type AttributeRef struct {
	ID   uuid.UUID
	Name string
}

type Product struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Description string
	Status      ProductStatus
	BasePrice   money.Money
	// CompareAtPrice is an optional "was" price shown struck through next to
	// BasePrice to indicate a discount — nil means the product isn't on sale.
	CompareAtPrice *money.Money
	CategoryIDs    []uuid.UUID
	CatalogIDs     []uuid.UUID
	Attributes     []AttributeRef
	// VariantCount is always populated (cheap to compute). Variants itself
	// is only populated by FindByID — List skips loading full variant data
	// for performance, since the admin table only needs the count.
	VariantCount int
	Variants     []ProductVariant
	// PrimaryMedia is the first media item (by position), populated cheaply
	// in both List and FindByID for storefront thumbnails — nil if the
	// product has no media yet.
	PrimaryMedia *ProductMedia
	Media        []ProductMedia
	// InStock is true when the product has no variants yet (inventory isn't
	// tracked until a variant exists) or at least one variant has available
	// stock. Computed from the inventory module's data, not persisted here.
	InStock   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
