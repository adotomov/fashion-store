package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
)

type ProductTypeService struct {
	repo ProductTypeRepository
}

func NewProductTypeService(repo ProductTypeRepository) *ProductTypeService {
	return &ProductTypeService{repo: repo}
}

func (s *ProductTypeService) CreateProductType(ctx context.Context, input CreateProductTypeInput) (*domain.ProductType, error) {
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

		productType, err := s.repo.Create(ctx, domain.ProductType{Name: input.Name, Slug: slug})
		if err == nil {
			return productType, nil
		}
		if !errors.Is(err, ErrSlugConflict) {
			return nil, err
		}
		lastErr = err
	}

	return nil, lastErr
}

func (s *ProductTypeService) ListProductTypes(ctx context.Context) ([]domain.ProductType, error) {
	return s.repo.List(ctx)
}

func (s *ProductTypeService) GetProductType(ctx context.Context, id uuid.UUID) (*domain.ProductType, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ProductTypeService) UpdateProductType(ctx context.Context, id uuid.UUID, input UpdateProductTypeInput) (*domain.ProductType, error) {
	productType, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		if *input.Name == "" {
			return nil, domain.ValidationError("name is required")
		}
		productType.Name = *input.Name
	}
	if input.Position != nil {
		productType.Position = *input.Position
	}

	return s.repo.Update(ctx, *productType)
}

func (s *ProductTypeService) DeleteProductType(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
