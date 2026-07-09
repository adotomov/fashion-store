package application

import (
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type CreateCatalogInput struct {
	Name string
}

type UpdateCatalogInput struct {
	Name        *string
	Description *string
	Status      *domain.Status
}

type CreateCategoryInput struct {
	Name               string
	ParentID           *uuid.UUID
	ProductTypeID      uuid.UUID
	InternalIdentifier string
}

type UpdateCategoryInput struct {
	Name               *string
	ParentID           *uuid.UUID
	ProductTypeID      *uuid.UUID
	InternalIdentifier *string
}

type CreateProductTypeInput struct {
	Name string
}

type UpdateProductTypeInput struct {
	Name     *string
	Position *int
}

type CreateAttributeInput struct {
	Name string
}

type CreateProductInput struct {
	Name string
}

type UpdateProductInput struct {
	Name                *string
	Description         *string
	Status              *domain.ProductStatus
	BasePrice           *money.Money
	CompareAtPrice      *money.Money
	ClearCompareAtPrice bool
	TaxGroupID          *uuid.UUID
	ClearTaxGroupID     bool
}

// TopProduct is one row of the admin dashboard's "best sellers" ranking —
// products ordered by total quantity sold across all orders.
type TopProduct struct {
	ProductID    uuid.UUID
	ProductName  string
	QuantitySold int
	OrderCount   int
}

// CatalogStats summarizes catalog composition for the admin dashboard.
type CatalogStats struct {
	TotalProducts    int
	ActiveProducts   int
	DraftProducts    int
	ArchivedProducts int
	TotalVariants    int
	TotalCategories  int
	TopProducts      []TopProduct
}

type CreateVariantInput struct {
	AttributeValueIDs []uuid.UUID
	PriceOverride     *money.Money
}

type UpdateVariantInput struct {
	// AttributeValueIDs always replaces the variant's full attribute
	// composition — the editor UI re-submits the complete set on every save.
	AttributeValueIDs  []uuid.UUID
	PriceOverride      *money.Money
	ClearPriceOverride bool
}

type CreateMediaInput struct {
	Bucket      string
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	Position    int
	AltText     string
}

type UpdateMediaInput struct {
	Position *int
	AltText  *string
}
