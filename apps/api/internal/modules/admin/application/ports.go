package application

import (
	"context"
	"io"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
)

// MediaStorage isolates the GCS-compatible object storage vendor from
// application logic. Implemented in internal/platform/storage — same port
// the catalog module uses for product media and category thumbnails.
type MediaStorage interface {
	EnsureBucket(ctx context.Context, bucket string) error
	Upload(ctx context.Context, bucket, objectKey, contentType string, content io.Reader) (sizeBytes int64, err error)
	Open(ctx context.Context, bucket, objectKey string) (reader io.ReadCloser, contentType string, err error)
	Delete(ctx context.Context, bucket, objectKey string) error
}

// StoreSettingsRepository persists the single store_settings row. Get
// returns that row (it always exists — seeded by migration); Update
// persists changes to it.
type StoreSettingsRepository interface {
	Get(ctx context.Context) (*domain.StoreSettings, error)
	Update(ctx context.Context, settings domain.StoreSettings) (*domain.StoreSettings, error)
}

// StoreAddressRepository persists the zero-or-more addresses for the store
// settings singleton — multi-location stores have more than one.
type StoreAddressRepository interface {
	List(ctx context.Context, storeSettingsID uuid.UUID) ([]domain.StoreAddress, error)
	Create(ctx context.Context, address domain.StoreAddress) (*domain.StoreAddress, error)
	Update(ctx context.Context, address domain.StoreAddress) (*domain.StoreAddress, error)
	Delete(ctx context.Context, id uuid.UUID) error
	// ClearDefault unsets is_default on every address for the store, used
	// before setting a new one so exactly one default ever exists.
	ClearDefault(ctx context.Context, storeSettingsID uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*domain.StoreAddress, error)
}

// StoreDocumentRepository persists legal documents keyed by (type, locale).
// Documents are either inline Markdown (UpsertContent) or GCS file uploads (Upsert).
type StoreDocumentRepository interface {
	List(ctx context.Context, docType domain.DocumentType) ([]domain.StoreDocument, error)
	Get(ctx context.Context, docType domain.DocumentType, locale string) (*domain.StoreDocument, error)
	Upsert(ctx context.Context, doc domain.StoreDocument) (*domain.StoreDocument, error)
	UpsertContent(ctx context.Context, docType domain.DocumentType, locale, content string) (*domain.StoreDocument, error)
	Delete(ctx context.Context, docType domain.DocumentType, locale string) error
}

// HeroSettingsRepository persists the singleton hero_settings row.
// The row always exists (seeded by migration) and is only ever upserted.
type HeroSettingsRepository interface {
	GetHeroSettings(ctx context.Context) (domain.HeroSettings, error)
	SaveHeroSettings(ctx context.Context, s domain.HeroSettings) (domain.HeroSettings, error)
}

// EditorialBannerRepository persists the singleton editorial_banner_settings row.
// Like hero_settings, the row always exists (seeded by migration) and is only
// ever upserted.
type EditorialBannerRepository interface {
	GetEditorialBanner(ctx context.Context) (domain.EditorialBanner, error)
	SaveEditorialBanner(ctx context.Context, b domain.EditorialBanner) (domain.EditorialBanner, error)
}

// TranslationGateway is the narrow slice of the i18n module's translation
// service the store-settings service uses to store per-locale overrides for the
// editorial home-page content (hero, editorial banner, home sections). Its
// signatures match *i18n/application.TranslationService, which is wired as the
// adapter in modules.go. The base English text lives in the settings rows
// themselves; every other locale's text lives in translation rows keyed by a
// stable synthetic UUID per singleton/section (see store_settings_i18n.go).
type TranslationGateway interface {
	Get(ctx context.Context, entityType string, entityID uuid.UUID, locale string) (map[string]string, error)
	Set(ctx context.Context, entityType string, entityID uuid.UUID, locale, field, value string) error
}

// HomeSectionsRepository persists the home_sections and home_section_products
// rows. The four section rows are seeded by migration and are only updated,
// never inserted or deleted from application code.
type HomeSectionsRepository interface {
	ListHomeSections(ctx context.Context) ([]domain.HomeSection, error)
	SaveHomeSection(ctx context.Context, s domain.HomeSection) (domain.HomeSection, error)
	GetSectionProductIDs(ctx context.Context, sectionID string) ([]uuid.UUID, error)
	SetSectionProducts(ctx context.Context, sectionID string, productIDs []uuid.UUID) error
	GetSectionCategoryGroups(ctx context.Context, sectionID string) ([]domain.SectionCategoryGroup, error)
	SetSectionCategoryGroups(ctx context.Context, sectionID string, groups []domain.SectionCategoryGroup) error
}
