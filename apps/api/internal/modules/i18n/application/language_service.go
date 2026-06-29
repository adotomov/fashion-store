package application

import (
	"context"
	"strings"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/domain"
)

type LanguageRepository interface {
	List(ctx context.Context) ([]domain.Language, error)
	Get(ctx context.Context, code string) (*domain.Language, error)
	Create(ctx context.Context, lang domain.Language) (*domain.Language, error)
	SetEnabled(ctx context.Context, code string, enabled bool) (*domain.Language, error)
	Delete(ctx context.Context, code string) error
}

type LanguageService struct {
	repo LanguageRepository
}

func NewLanguageService(repo LanguageRepository) *LanguageService {
	return &LanguageService{repo: repo}
}

func (s *LanguageService) List(ctx context.Context) ([]domain.Language, error) {
	return s.repo.List(ctx)
}

func (s *LanguageService) ListEnabled(ctx context.Context) ([]domain.Language, error) {
	all, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	enabled := make([]domain.Language, 0, len(all))
	for _, l := range all {
		if l.Enabled {
			enabled = append(enabled, l)
		}
	}
	return enabled, nil
}

func (s *LanguageService) Add(ctx context.Context, code, name string) (*domain.Language, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	if code == "" || name == "" {
		return nil, domain.ErrInvalidLanguageCode
	}
	if existing, err := s.repo.Get(ctx, code); err == nil && existing != nil {
		return nil, domain.ErrLanguageAlreadyExists
	}
	return s.repo.Create(ctx, domain.Language{Code: code, Name: name, IsDefault: false, Enabled: true})
}

func (s *LanguageService) SetEnabled(ctx context.Context, code string, enabled bool) (*domain.Language, error) {
	lang, err := s.repo.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	if lang.IsDefault && !enabled {
		return nil, domain.ErrCannotModifyDefaultLocale
	}
	return s.repo.SetEnabled(ctx, code, enabled)
}

func (s *LanguageService) Delete(ctx context.Context, code string) error {
	lang, err := s.repo.Get(ctx, code)
	if err != nil {
		return err
	}
	if lang.IsDefault {
		return domain.ErrCannotModifyDefaultLocale
	}
	return s.repo.Delete(ctx, code)
}
