package application

import (
	"context"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/domain"
)

type UIStringRepository interface {
	// ListAll returns every key with its value in every locale it has been
	// translated into, plus the English baseline — backs the admin grid.
	ListAll(ctx context.Context) ([]domain.UIString, error)
	GetByLocale(ctx context.Context, locale string) (map[string]string, error)
	Upsert(ctx context.Context, s domain.UIString) error
	// SeedDefaults inserts English baseline keys that don't already exist —
	// idempotent, safe to call on every startup as new keys are added in code.
	SeedDefaults(ctx context.Context, defaults map[string]string) error
}

type UIStringService struct {
	repo UIStringRepository
}

func NewUIStringService(repo UIStringRepository) *UIStringService {
	return &UIStringService{repo: repo}
}

func (s *UIStringService) ListAll(ctx context.Context) ([]domain.UIString, error) {
	return s.repo.ListAll(ctx)
}

// GetByLocale returns the full string map for a locale, falling back to the
// English value for any key the locale hasn't translated yet.
func (s *UIStringService) GetByLocale(ctx context.Context, locale string) (map[string]string, error) {
	english, err := s.repo.GetByLocale(ctx, domain.DefaultLocale)
	if err != nil {
		return nil, err
	}
	if locale == domain.DefaultLocale {
		return english, nil
	}

	translated, err := s.repo.GetByLocale(ctx, locale)
	if err != nil {
		return nil, err
	}

	merged := make(map[string]string, len(english))
	for k, v := range english {
		merged[k] = v
	}
	for k, v := range translated {
		merged[k] = v
	}
	return merged, nil
}

func (s *UIStringService) Upsert(ctx context.Context, key, locale, value string) error {
	return s.repo.Upsert(ctx, domain.UIString{Key: key, Locale: locale, Value: value})
}

func (s *UIStringService) SeedDefaults(ctx context.Context, defaults map[string]string) error {
	return s.repo.SeedDefaults(ctx, defaults)
}
