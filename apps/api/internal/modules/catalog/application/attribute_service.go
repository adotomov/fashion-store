package application

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
)

// hexColorPattern accepts 3- or 6-digit CSS hex colors (e.g. #B2543C, #abc).
var hexColorPattern = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

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
	attribute, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if attribute.IsSystem {
		return domain.ErrSystemAttributeReadOnly
	}
	return s.repo.Delete(ctx, id)
}

// AddValue appends a value to an attribute. For color-typed attributes the
// value is the color's name and colorHex is the picked palette color (which
// is required and validated); for text attributes colorHex is ignored.
func (s *AttributeService) AddValue(ctx context.Context, attributeID uuid.UUID, value string, colorHex *string) (*domain.AttributeValue, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, domain.ValidationError("value is required")
	}

	attribute, err := s.repo.FindByID(ctx, attributeID)
	if err != nil {
		return nil, err
	}

	var normalized *string
	if attribute.Type == domain.AttributeTypeColor {
		if colorHex == nil || strings.TrimSpace(*colorHex) == "" {
			return nil, domain.ValidationError("color is required for color attributes")
		}
		hex := strings.TrimSpace(*colorHex)
		if !hexColorPattern.MatchString(hex) {
			return nil, domain.ValidationError("color must be a valid hex value, e.g. #B2543C")
		}
		normalized = &hex
	}

	return s.repo.AddValue(ctx, attributeID, value, normalized)
}

func (s *AttributeService) DeleteValue(ctx context.Context, attributeID, valueID uuid.UUID) error {
	return s.repo.DeleteValue(ctx, attributeID, valueID)
}
