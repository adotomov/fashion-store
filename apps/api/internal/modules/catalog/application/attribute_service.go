package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
)

type AttributeService struct {
	repo AttributeRepository
}

func NewAttributeService(repo AttributeRepository) *AttributeService {
	return &AttributeService{repo: repo}
}

func (s *AttributeService) CreateAttribute(ctx context.Context, input CreateAttributeInput) (*domain.Attribute, error) {
	if input.Name == "" {
		return nil, domain.ValidationError("name is required")
	}
	return s.repo.Create(ctx, domain.Attribute{Name: input.Name})
}

func (s *AttributeService) ListAttributes(ctx context.Context) ([]domain.Attribute, error) {
	return s.repo.List(ctx)
}

func (s *AttributeService) GetAttribute(ctx context.Context, id uuid.UUID) (*domain.Attribute, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *AttributeService) UpdateAttribute(ctx context.Context, id uuid.UUID, name string) (*domain.Attribute, error) {
	if name == "" {
		return nil, domain.ValidationError("name is required")
	}
	return s.repo.UpdateName(ctx, id, name)
}

func (s *AttributeService) DeleteAttribute(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *AttributeService) AddValue(ctx context.Context, attributeID uuid.UUID, value string) (*domain.AttributeValue, error) {
	if value == "" {
		return nil, domain.ValidationError("value is required")
	}
	return s.repo.AddValue(ctx, attributeID, value)
}

func (s *AttributeService) DeleteValue(ctx context.Context, attributeID, valueID uuid.UUID) error {
	return s.repo.DeleteValue(ctx, attributeID, valueID)
}
