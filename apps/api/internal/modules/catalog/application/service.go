package application

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
)

const maxSlugAttempts = 5

type CatalogService struct {
	repo CatalogRepository
}

func NewCatalogService(repo CatalogRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

func (s *CatalogService) CreateCatalog(ctx context.Context, input CreateCatalogInput) (*domain.Catalog, error) {
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

		catalog, err := s.repo.Create(ctx, domain.Catalog{
			Name:   input.Name,
			Slug:   slug,
			Status: domain.StatusDraft,
		})
		if err == nil {
			return catalog, nil
		}
		if !errors.Is(err, ErrSlugConflict) {
			return nil, err
		}
		lastErr = err
	}

	return nil, lastErr
}

func (s *CatalogService) ListCatalogs(ctx context.Context) ([]domain.Catalog, error) {
	return s.repo.List(ctx)
}

func (s *CatalogService) GetCatalog(ctx context.Context, id uuid.UUID) (*domain.Catalog, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *CatalogService) UpdateCatalog(ctx context.Context, id uuid.UUID, input UpdateCatalogInput) (*domain.Catalog, error) {
	catalog, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		if *input.Name == "" {
			return nil, domain.ValidationError("name is required")
		}
		catalog.Name = *input.Name
	}
	if input.Description != nil {
		catalog.Description = *input.Description
	}
	if input.Status != nil {
		if !input.Status.Valid() {
			return nil, domain.ErrInvalidStatus
		}
		catalog.Status = *input.Status
	}

	return s.repo.Update(ctx, *catalog)
}

func (s *CatalogService) DeleteCatalog(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func randomSuffix() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 5)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
