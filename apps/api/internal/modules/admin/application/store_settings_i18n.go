package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
)

// Per-locale translation of the admin-managed editorial home-page content
// (hero, editorial banner, home sections). The base English copy lives in the
// settings rows; every other locale's copy is layered on top via the shared
// i18n translations table, exactly like catalog entities. Only human-readable
// text is translated — URLs, image references, and enabled flags are the same
// in every language and always come from the base row.

// Entity-type tags stored in the translations table's entity_type column.
const (
	translationEntityHero            = "hero"
	translationEntityEditorialBanner = "editorial_banner"
	translationEntityHomeSection     = "home_section"

	// defaultTranslationLocale is the base language whose content lives directly
	// in the settings rows (mirrors i18n/domain.DefaultLocale, kept local to
	// avoid a cross-module dependency for a single constant).
	defaultTranslationLocale = "en"
)

// The singletons and each home section need a stable UUID to key their
// translation rows (the shared table is keyed by uuid). We derive them
// deterministically from a fixed namespace so they never collide with real
// entity IDs and never have to be hardcoded or migrated.
var translationNamespace = uuid.MustParse("6f8d4e2a-1c3b-4a5d-9e7f-0a1b2c3d4e5f")

var (
	heroTranslationID            = uuid.NewSHA1(translationNamespace, []byte("hero"))
	editorialBannerTranslationID = uuid.NewSHA1(translationNamespace, []byte("editorial_banner"))
)

// homeSectionTranslationID maps a home-section id (e.g. "spotlights") to its
// stable translation-row UUID.
func homeSectionTranslationID(sectionID string) uuid.UUID {
	return uuid.NewSHA1(translationNamespace, []byte("home_section:"+sectionID))
}

// Translatable field names — must match the JSON field names the admin/storefront
// handlers use so the frontend can round-trip them.
var (
	heroTranslatableFields      = []string{"eyebrow", "heading", "subtext", "cta_primary_label", "cta_secondary_label"}
	editorialTranslatableFields = []string{"eyebrow", "heading", "subtext", "cta_label"}
	homeSectionFields           = []string{"eyebrow", "heading"}
)

// overlay returns the translated value for a field when present and non-empty,
// otherwise the base English value.
func overlay(base string, tr map[string]string, field string) string {
	if v, ok := tr[field]; ok && v != "" {
		return v
	}
	return base
}

// filterFields keeps only the allowed keys from a submitted translation map, so
// callers can't write arbitrary fields into the translations table.
func filterFields(fields map[string]string, allowed []string) map[string]string {
	out := make(map[string]string, len(allowed))
	for _, f := range allowed {
		if v, ok := fields[f]; ok {
			out[f] = v
		}
	}
	return out
}

// ─── Hero ───────────────────────────────────────────────────────────────────

// LocalizedHeroSettings returns the hero with its text fields overlaid for the
// given locale (English base for anything untranslated). Used by the storefront.
func (s *StoreSettingsService) LocalizedHeroSettings(ctx context.Context, locale string) (domain.HeroSettings, error) {
	hero, err := s.heroRepo.GetHeroSettings(ctx)
	if err != nil {
		return hero, err
	}
	if s.translations == nil || locale == "" || locale == defaultTranslationLocale {
		return hero, nil
	}
	tr, err := s.translations.Get(ctx, translationEntityHero, heroTranslationID, locale)
	if err != nil {
		return hero, err
	}
	hero.Eyebrow = overlay(hero.Eyebrow, tr, "eyebrow")
	hero.Heading = overlay(hero.Heading, tr, "heading")
	hero.Subtext = overlay(hero.Subtext, tr, "subtext")
	hero.CTAPrimaryLabel = overlay(hero.CTAPrimaryLabel, tr, "cta_primary_label")
	if hero.CTASecondaryLabel != nil {
		v := overlay(*hero.CTASecondaryLabel, tr, "cta_secondary_label")
		hero.CTASecondaryLabel = &v
	}
	return hero, nil
}

// HeroTranslations returns the raw per-locale text overrides (empty when unset),
// for the admin editor of a non-default language.
func (s *StoreSettingsService) HeroTranslations(ctx context.Context, locale string) (map[string]string, error) {
	if s.translations == nil {
		return map[string]string{}, nil
	}
	return s.translations.Get(ctx, translationEntityHero, heroTranslationID, locale)
}

// SaveHeroTranslations stores the hero text overrides for a non-default locale.
func (s *StoreSettingsService) SaveHeroTranslations(ctx context.Context, locale string, fields map[string]string) error {
	return s.saveTranslations(ctx, translationEntityHero, heroTranslationID, locale, filterFields(fields, heroTranslatableFields))
}

// ─── Editorial banner ───────────────────────────────────────────────────────

func (s *StoreSettingsService) LocalizedEditorialBanner(ctx context.Context, locale string) (domain.EditorialBanner, error) {
	banner, err := s.bannerRepo.GetEditorialBanner(ctx)
	if err != nil {
		return banner, err
	}
	if s.translations == nil || locale == "" || locale == defaultTranslationLocale {
		return banner, nil
	}
	tr, err := s.translations.Get(ctx, translationEntityEditorialBanner, editorialBannerTranslationID, locale)
	if err != nil {
		return banner, err
	}
	banner.Eyebrow = overlay(banner.Eyebrow, tr, "eyebrow")
	banner.Heading = overlay(banner.Heading, tr, "heading")
	banner.Subtext = overlay(banner.Subtext, tr, "subtext")
	banner.CTALabel = overlay(banner.CTALabel, tr, "cta_label")
	return banner, nil
}

func (s *StoreSettingsService) EditorialBannerTranslations(ctx context.Context, locale string) (map[string]string, error) {
	if s.translations == nil {
		return map[string]string{}, nil
	}
	return s.translations.Get(ctx, translationEntityEditorialBanner, editorialBannerTranslationID, locale)
}

func (s *StoreSettingsService) SaveEditorialBannerTranslations(ctx context.Context, locale string, fields map[string]string) error {
	return s.saveTranslations(ctx, translationEntityEditorialBanner, editorialBannerTranslationID, locale, filterFields(fields, editorialTranslatableFields))
}

// ─── Home sections ──────────────────────────────────────────────────────────

// LocalizedHomeSections returns all home sections with their eyebrow/heading
// overlaid for the given locale. Used by the storefront.
func (s *StoreSettingsService) LocalizedHomeSections(ctx context.Context, locale string) ([]domain.HomeSection, error) {
	sections, err := s.homeSectionsRepo.ListHomeSections(ctx)
	if err != nil {
		return nil, err
	}
	if s.translations == nil || locale == "" || locale == defaultTranslationLocale {
		return sections, nil
	}
	for i := range sections {
		tr, err := s.translations.Get(ctx, translationEntityHomeSection, homeSectionTranslationID(sections[i].ID), locale)
		if err != nil {
			return nil, err
		}
		sections[i].Eyebrow = overlay(sections[i].Eyebrow, tr, "eyebrow")
		sections[i].Heading = overlay(sections[i].Heading, tr, "heading")
	}
	return sections, nil
}

func (s *StoreSettingsService) HomeSectionTranslations(ctx context.Context, sectionID, locale string) (map[string]string, error) {
	if s.translations == nil {
		return map[string]string{}, nil
	}
	return s.translations.Get(ctx, translationEntityHomeSection, homeSectionTranslationID(sectionID), locale)
}

func (s *StoreSettingsService) SaveHomeSectionTranslations(ctx context.Context, sectionID, locale string, fields map[string]string) error {
	return s.saveTranslations(ctx, translationEntityHomeSection, homeSectionTranslationID(sectionID), locale, filterFields(fields, homeSectionFields))
}

// saveTranslations upserts each provided field for one entity/locale. An empty
// value clears that field's override (handled by the gateway).
func (s *StoreSettingsService) saveTranslations(ctx context.Context, entityType string, entityID uuid.UUID, locale string, fields map[string]string) error {
	if s.translations == nil {
		return nil
	}
	for field, value := range fields {
		if err := s.translations.Set(ctx, entityType, entityID, locale, field, value); err != nil {
			return err
		}
	}
	return nil
}
