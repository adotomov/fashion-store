package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/domain"
)

type TranslationRepository interface {
	// Get returns field->value for one entity in one locale.
	Get(ctx context.Context, entityType string, entityID uuid.UUID, locale string) (map[string]string, error)
	// ListByEntityType returns entityID->field->value for every entity of a
	// type in one locale — used to overlay a whole listing page in one query.
	ListByEntityType(ctx context.Context, entityType string, locale string) (map[uuid.UUID]map[string]string, error)
	Upsert(ctx context.Context, t domain.Translation) error
	Delete(ctx context.Context, entityType string, entityID uuid.UUID, locale, field string) error
	// DeleteEntity removes every translation row for an entity, regardless of
	// locale or field — used when the underlying entity is deleted.
	DeleteEntity(ctx context.Context, entityType string, entityID uuid.UUID) error
}

type TranslationService struct {
	repo TranslationRepository
}

func NewTranslationService(repo TranslationRepository) *TranslationService {
	return &TranslationService{repo: repo}
}

func (s *TranslationService) Get(ctx context.Context, entityType string, entityID uuid.UUID, locale string) (map[string]string, error) {
	if locale == domain.DefaultLocale {
		return map[string]string{}, nil
	}
	return s.repo.Get(ctx, entityType, entityID, locale)
}

func (s *TranslationService) ListByEntityType(ctx context.Context, entityType string, locale string) (map[uuid.UUID]map[string]string, error) {
	if locale == domain.DefaultLocale {
		return map[uuid.UUID]map[string]string{}, nil
	}
	return s.repo.ListByEntityType(ctx, entityType, locale)
}

func (s *TranslationService) Set(ctx context.Context, entityType string, entityID uuid.UUID, locale, field, value string) error {
	if locale == domain.DefaultLocale {
		return domain.ErrCannotModifyDefaultLocale
	}
	if value == "" {
		return s.repo.Delete(ctx, entityType, entityID, locale, field)
	}
	return s.repo.Upsert(ctx, domain.Translation{
		EntityType: entityType,
		EntityID:   entityID,
		Locale:     locale,
		Field:      field,
		Value:      value,
	})
}

func (s *TranslationService) DeleteEntity(ctx context.Context, entityType string, entityID uuid.UUID) error {
	return s.repo.DeleteEntity(ctx, entityType, entityID)
}
