package application

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type ProductService struct {
	repo    ProductRepository
	storage MediaStorage
	bucket  string
}

func NewProductService(repo ProductRepository, storage MediaStorage, bucket string) *ProductService {
	return &ProductService{repo: repo, storage: storage, bucket: bucket}
}

// CreateProduct creates a product with just a name, matching the
// create-modal-then-edit pattern used for catalogs and categories. Status
// defaults to draft and the price defaults to zero until set via
// UpdateProduct on the product's editor page.
func (s *ProductService) CreateProduct(ctx context.Context, input CreateProductInput) (*domain.Product, error) {
	if input.Name == "" {
		return nil, domain.ValidationError("name is required")
	}

	base := slugify(input.Name)

	var lastErr error
	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		slug := base
		if attempt > 0 {
			slug = fmt.Sprintf("%s-%s", base, randomSuffix())
		}

		product, err := s.repo.Create(ctx, domain.Product{
			Name:      input.Name,
			Slug:      slug,
			Status:    domain.ProductStatusDraft,
			BasePrice: money.Money{AmountMinor: 0, Currency: "EUR"},
		})
		if err == nil {
			return product, nil
		}
		if !errors.Is(err, ErrSlugConflict) {
			return nil, err
		}
		lastErr = err
	}

	return nil, lastErr
}

func (s *ProductService) ListProducts(ctx context.Context) ([]domain.Product, error) {
	return s.repo.List(ctx)
}

func (s *ProductService) GetProduct(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ProductService) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	return s.repo.FindBySlug(ctx, slug)
}

func (s *ProductService) UpdateProduct(ctx context.Context, id uuid.UUID, input UpdateProductInput) (*domain.Product, error) {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		if *input.Name == "" {
			return nil, domain.ValidationError("name is required")
		}
		product.Name = *input.Name
	}
	if input.Description != nil {
		product.Description = *input.Description
	}
	if input.Status != nil {
		if !input.Status.Valid() {
			return nil, domain.ErrInvalidStatus
		}
		product.Status = *input.Status
	}
	if input.BasePrice != nil {
		product.BasePrice = *input.BasePrice
	}
	if input.ClearCompareAtPrice {
		product.CompareAtPrice = nil
	} else if input.CompareAtPrice != nil {
		product.CompareAtPrice = input.CompareAtPrice
	}

	return s.repo.Update(ctx, *product)
}

func (s *ProductService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *ProductService) SetCategories(ctx context.Context, productID uuid.UUID, categoryIDs []uuid.UUID) error {
	return s.repo.SetCategories(ctx, productID, categoryIDs)
}

func (s *ProductService) SetCatalogs(ctx context.Context, productID uuid.UUID, catalogIDs []uuid.UUID) error {
	return s.repo.SetCatalogs(ctx, productID, catalogIDs)
}

func (s *ProductService) SetAttributes(ctx context.Context, productID uuid.UUID, attributeIDs []uuid.UUID) error {
	return s.repo.SetAttributes(ctx, productID, attributeIDs)
}

func (s *ProductService) ProductIDsByCategory(ctx context.Context, categoryID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.ProductIDsByCategory(ctx, categoryID)
}

func (s *ProductService) ProductIDsByCatalog(ctx context.Context, catalogID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.ProductIDsByCatalog(ctx, catalogID)
}

func (s *ProductService) ProductIDsByAttributeValues(ctx context.Context, valueIDs []uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.ProductIDsByAttributeValues(ctx, valueIDs)
}

func (s *ProductService) AttributeFacets(ctx context.Context, categoryIDs []uuid.UUID, catalogID *uuid.UUID) ([]domain.AttributeFacet, error) {
	return s.repo.AttributeFacets(ctx, categoryIDs, catalogID)
}

func (s *ProductService) AddVariant(ctx context.Context, productID uuid.UUID, input CreateVariantInput) (*domain.ProductVariant, error) {
	if len(input.AttributeValueIDs) == 0 {
		return nil, domain.ValidationError("at least one attribute value is required to define a variant")
	}

	variant := domain.ProductVariant{
		ProductID:     productID,
		PriceOverride: input.PriceOverride,
	}
	return s.repo.CreateVariant(ctx, variant, input.AttributeValueIDs)
}

func (s *ProductService) UpdateVariant(ctx context.Context, variantID uuid.UUID, input UpdateVariantInput) (*domain.ProductVariant, error) {
	if len(input.AttributeValueIDs) == 0 {
		return nil, domain.ValidationError("at least one attribute value is required to define a variant")
	}

	variant, err := s.repo.FindVariantByID(ctx, variantID)
	if err != nil {
		return nil, err
	}

	if input.ClearPriceOverride {
		variant.PriceOverride = nil
	} else if input.PriceOverride != nil {
		variant.PriceOverride = input.PriceOverride
	}

	return s.repo.UpdateVariant(ctx, *variant, input.AttributeValueIDs)
}

func (s *ProductService) DeleteVariant(ctx context.Context, variantID uuid.UUID) error {
	return s.repo.DeleteVariant(ctx, variantID)
}

// UploadMedia streams content to object storage and registers the
// resulting object as product media in one call — the admin UI never talks
// to the storage backend directly (its self-signed local cert would break
// browser <img> requests anyway), only through this server-mediated path.
func (s *ProductService) UploadMedia(ctx context.Context, productID uuid.UUID, filename, contentType string, content io.Reader, position int, altText string) (*domain.ProductMedia, error) {
	if err := s.storage.EnsureBucket(ctx, s.bucket); err != nil {
		return nil, err
	}

	objectKey := fmt.Sprintf("%s/%s-%s", productID, uuid.NewString(), filename)

	sizeBytes, err := s.storage.Upload(ctx, s.bucket, objectKey, contentType, content)
	if err != nil {
		return nil, err
	}

	return s.repo.CreateMedia(ctx, domain.ProductMedia{
		ProductID:   productID,
		Bucket:      s.bucket,
		ObjectKey:   objectKey,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		Position:    position,
		AltText:     altText,
	})
}

// OpenMedia streams a stored media object back out, for the
// backend-proxied media-serving endpoint.
func (s *ProductService) OpenMedia(ctx context.Context, mediaID uuid.UUID) (io.ReadCloser, string, error) {
	media, err := s.repo.FindMediaByID(ctx, mediaID)
	if err != nil {
		return nil, "", err
	}
	return s.storage.Open(ctx, media.Bucket, media.ObjectKey)
}

func (s *ProductService) UpdateMedia(ctx context.Context, mediaID uuid.UUID, input UpdateMediaInput) (*domain.ProductMedia, error) {
	media, err := s.repo.FindMediaByID(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	if input.Position != nil {
		media.Position = *input.Position
	}
	if input.AltText != nil {
		media.AltText = *input.AltText
	}

	return s.repo.UpdateMedia(ctx, *media)
}

func (s *ProductService) CatalogStats(ctx context.Context) (CatalogStats, error) {
	return s.repo.Stats(ctx)
}

func (s *ProductService) DeleteMedia(ctx context.Context, mediaID uuid.UUID) error {
	media, err := s.repo.FindMediaByID(ctx, mediaID)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteMedia(ctx, mediaID); err != nil {
		return err
	}

	// Best-effort: the DB row is the source of truth, so a storage cleanup
	// failure here shouldn't surface as an API error.
	_ = s.storage.Delete(ctx, media.Bucket, media.ObjectKey)
	return nil
}
