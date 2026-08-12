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
	SetDefault(ctx context.Context, code string) (*domain.Language, error)
	SetCountry(ctx context.Context, code, countryCode string) (*domain.Language, error)
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
	// The English base (the fallback for every other locale) and the current
	// default display language must never be turned off.
	if !enabled && (lang.Code == domain.DefaultLocale || lang.IsDefault) {
		return nil, domain.ErrCannotModifyDefaultLocale
	}
	return s.repo.SetEnabled(ctx, code, enabled)
}

// SetDefault makes an enabled language the store's default display language.
func (s *LanguageService) SetDefault(ctx context.Context, code string) (*domain.Language, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	lang, err := s.repo.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	if !lang.Enabled {
		return nil, domain.ErrLanguageNotEnabled
	}
	return s.repo.SetDefault(ctx, code)
}

// SetCountry sets (or clears, when empty) the ISO-3166 country a language is
// geo-served to, so a request's region can resolve to it.
func (s *LanguageService) SetCountry(ctx context.Context, code, countryCode string) (*domain.Language, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	if _, err := s.repo.Get(ctx, code); err != nil {
		return nil, err
	}
	return s.repo.SetCountry(ctx, code, countryCode)
}

func (s *LanguageService) Delete(ctx context.Context, code string) error {
	lang, err := s.repo.Get(ctx, code)
	if err != nil {
		return err
	}
	if lang.Code == domain.DefaultLocale || lang.IsDefault {
		return domain.ErrCannotModifyDefaultLocale
	}
	return s.repo.Delete(ctx, code)
}
