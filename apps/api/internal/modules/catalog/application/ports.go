package application

import (
	"context"
	"errors"
	"io"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
)

// MediaStorage isolates the GCS-compatible object storage vendor from
// application logic. Implemented in internal/platform/storage.
type MediaStorage interface {
	EnsureBucket(ctx context.Context, bucket string) error
	Upload(ctx context.Context, bucket, objectKey, contentType string, content io.Reader) (sizeBytes int64, err error)
	Open(ctx context.Context, bucket, objectKey string) (reader io.ReadCloser, contentType string, err error)
	Delete(ctx context.Context, bucket, objectKey string) error
}

// ErrSlugConflict signals a unique-slug constraint violation so the service
// can retry with a different candidate slug.
var ErrSlugConflict = errors.New("slug already in use")

type CatalogRepository interface {
	Create(ctx context.Context, catalog domain.Catalog) (*domain.Catalog, error)
	List(ctx context.Context) ([]domain.Catalog, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Catalog, error)
	Update(ctx context.Context, catalog domain.Catalog) (*domain.Catalog, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type CategoryRepository interface {
	Create(ctx context.Context, category domain.Category) (*domain.Category, error)
	List(ctx context.Context) ([]domain.Category, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	Update(ctx context.Context, category domain.Category) (*domain.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ProductTypeRepository interface {
	Create(ctx context.Context, productType domain.ProductType) (*domain.ProductType, error)
	List(ctx context.Context) ([]domain.ProductType, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.ProductType, error)
	Update(ctx context.Context, productType domain.ProductType) (*domain.ProductType, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type AttributeRepository interface {
	Create(ctx context.Context, attribute domain.Attribute) (*domain.Attribute, error)
	List(ctx context.Context) ([]domain.Attribute, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Attribute, error)
	UpdateName(ctx context.Context, id uuid.UUID, name string) (*domain.Attribute, error)
	Delete(ctx context.Context, id uuid.UUID) error
	AddValue(ctx context.Context, attributeID uuid.UUID, value string, colorHex *string) (*domain.AttributeValue, error)
	DeleteValue(ctx context.Context, attributeID, valueID uuid.UUID) error
}

// ProductRepository persists products and their nested sub-resources
// (variants, media, category/catalog/attribute assignments) — nested the
// same way users.Repository nests addresses, since none of these have an
// independent lifecycle outside their parent product.
type ProductRepository interface {
	Create(ctx context.Context, product domain.Product) (*domain.Product, error)
	List(ctx context.Context) ([]domain.Product, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	FindBySlug(ctx context.Context, slug string) (*domain.Product, error)
	Update(ctx context.Context, product domain.Product) (*domain.Product, error)
	Delete(ctx context.Context, id uuid.UUID) error

	SetCategories(ctx context.Context, productID uuid.UUID, categoryIDs []uuid.UUID) error
	SetCatalogs(ctx context.Context, productID uuid.UUID, catalogIDs []uuid.UUID) error
	SetAttributes(ctx context.Context, productID uuid.UUID, attributeIDs []uuid.UUID) error

	// ProductIDsByCategory/ByCatalog support storefront filtering — List
	// deliberately omits category/catalog assignments per product (loaded
	// only by FindByID), so filtering by membership needs its own query.
	ProductIDsByCategory(ctx context.Context, categoryID uuid.UUID) ([]uuid.UUID, error)
	ProductIDsByCatalog(ctx context.Context, catalogID uuid.UUID) ([]uuid.UUID, error)
	// BestInCategoryProductIDs returns one active product ID per category,
	// picking the most recently created product in each. Used to populate the
	// "Best in its Category" home page section.
	BestInCategoryProductIDs(ctx context.Context) ([]uuid.UUID, error)

	// ProductIDsByAttributeValues matches products with AND semantics across
	// distinct attributes and OR semantics within the same attribute (e.g.
	// (Size=S OR Size=M) AND Color=Blue) — a product qualifies if some
	// variant carries each required attribute's value.
	ProductIDsByAttributeValues(ctx context.Context, valueIDs []uuid.UUID) ([]uuid.UUID, error)

	// AttributeFacets lists the attribute/value combinations actually used
	// by variants of active products, optionally narrowed to a set of
	// categories (OR) and/or a catalog — drives the storefront filter panel.
	AttributeFacets(ctx context.Context, categoryIDs []uuid.UUID, catalogID *uuid.UUID) ([]domain.AttributeFacet, error)

	CreateVariant(ctx context.Context, variant domain.ProductVariant, attributeValueIDs []uuid.UUID) (*domain.ProductVariant, error)
	FindVariantByID(ctx context.Context, variantID uuid.UUID) (*domain.ProductVariant, error)
	UpdateVariant(ctx context.Context, variant domain.ProductVariant, attributeValueIDs []uuid.UUID) (*domain.ProductVariant, error)
	DeleteVariant(ctx context.Context, variantID uuid.UUID) error

	CreateMedia(ctx context.Context, media domain.ProductMedia) (*domain.ProductMedia, error)
	FindMediaByID(ctx context.Context, mediaID uuid.UUID) (*domain.ProductMedia, error)
	UpdateMedia(ctx context.Context, media domain.ProductMedia) (*domain.ProductMedia, error)
	DeleteMedia(ctx context.Context, mediaID uuid.UUID) error

	Stats(ctx context.Context) (CatalogStats, error)

	// GetTaxGroupID returns the product's assigned VAT tax group, or nil.
	GetTaxGroupID(ctx context.Context, productID uuid.UUID) (*uuid.UUID, error)
}
