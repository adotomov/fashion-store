package application

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
)

type CategoryService struct {
	repo    CategoryRepository
	storage MediaStorage
	bucket  string
}

func NewCategoryService(repo CategoryRepository, storage MediaStorage, bucket string) *CategoryService {
	return &CategoryService{repo: repo, storage: storage, bucket: bucket}
}

func (s *CategoryService) CreateCategory(ctx context.Context, input CreateCategoryInput) (*domain.Category, error) {
	if input.Name == "" {
		return nil, domain.ValidationError("name is required")
	}
	if input.ProductTypeID == uuid.Nil {
		return nil, domain.ValidationError("product_type_id is required")
	}

	base := slugify(input.Name)

	var lastErr error
	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		slug := base
		if attempt > 0 {
			slug = fmt.Sprintf("%s-%s", base, randomSuffix())
		}

		category, err := s.repo.Create(ctx, domain.Category{
			Name:               input.Name,
			Slug:               slug,
			ParentID:           input.ParentID,
			ProductTypeID:      input.ProductTypeID,
			InternalIdentifier: input.InternalIdentifier,
		})
		if err == nil {
			return category, nil
		}
		if !errors.Is(err, ErrSlugConflict) {
			return nil, err
		}
		lastErr = err
	}

	return nil, lastErr
}

func (s *CategoryService) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return s.repo.List(ctx)
}

func (s *CategoryService) GetCategory(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *CategoryService) UpdateCategory(ctx context.Context, id uuid.UUID, input UpdateCategoryInput) (*domain.Category, error) {
	category, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		if *input.Name == "" {
			return nil, domain.ValidationError("name is required")
		}
		category.Name = *input.Name
	}
	if input.ParentID != nil {
		category.ParentID = input.ParentID
	}
	if input.ProductTypeID != nil {
		if *input.ProductTypeID == uuid.Nil {
			return nil, domain.ValidationError("product_type_id is required")
		}
		category.ProductTypeID = *input.ProductTypeID
	}
	if input.InternalIdentifier != nil {
		category.InternalIdentifier = *input.InternalIdentifier
	}

	return s.repo.Update(ctx, *category)
}

func (s *CategoryService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// UploadThumbnail streams content to object storage and records the
// resulting object on the category, mirroring ProductService.UploadMedia —
// the admin UI never talks to the storage backend directly (its
// self-signed local cert would break browser <img> requests anyway), only
// through this server-mediated path.
func (s *CategoryService) UploadThumbnail(ctx context.Context, categoryID uuid.UUID, filename, contentType string, content io.Reader) (*domain.Category, error) {
	category, err := s.repo.FindByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	if err := s.storage.EnsureBucket(ctx, s.bucket); err != nil {
		return nil, err
	}

	objectKey := fmt.Sprintf("categories/%s/%s-%s", categoryID, uuid.NewString(), filename)
	sizeBytes, err := s.storage.Upload(ctx, s.bucket, objectKey, contentType, content)
	if err != nil {
		return nil, err
	}

	bucket := s.bucket
	category.ThumbnailBucket = &bucket
	category.ThumbnailObjectKey = &objectKey
	category.ThumbnailContentType = &contentType
	category.ThumbnailSizeBytes = &sizeBytes

	return s.repo.Update(ctx, *category)
}

// OpenThumbnail streams a stored thumbnail object back out, for the
// backend-proxied thumbnail-serving endpoints (admin and storefront).
func (s *CategoryService) OpenThumbnail(ctx context.Context, categoryID uuid.UUID) (io.ReadCloser, string, error) {
	category, err := s.repo.FindByID(ctx, categoryID)
	if err != nil {
		return nil, "", err
	}
	if !category.HasThumbnail() {
		return nil, "", domain.ErrThumbnailNotFound
	}
	return s.storage.Open(ctx, *category.ThumbnailBucket, *category.ThumbnailObjectKey)
}

// DeleteThumbnail clears the category's thumbnail reference. Best-effort:
// the DB row is the source of truth, so a storage cleanup failure here
// shouldn't surface as an API error.
func (s *CategoryService) DeleteThumbnail(ctx context.Context, categoryID uuid.UUID) (*domain.Category, error) {
	category, err := s.repo.FindByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	if category.HasThumbnail() {
		_ = s.storage.Delete(ctx, *category.ThumbnailBucket, *category.ThumbnailObjectKey)
	}
	category.ThumbnailBucket = nil
	category.ThumbnailObjectKey = nil
	category.ThumbnailContentType = nil
	category.ThumbnailSizeBytes = nil

	return s.repo.Update(ctx, *category)
}
