package application

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
)

type StoreSettingsService struct {
	repo             StoreSettingsRepository
	heroRepo         HeroSettingsRepository
	bannerRepo       EditorialBannerRepository
	homeSectionsRepo HomeSectionsRepository
	storage          MediaStorage
	bucket           string
}

func NewStoreSettingsService(repo StoreSettingsRepository, storage MediaStorage, bucket string) *StoreSettingsService {
	// heroRepo is set separately via WithHeroRepo when the same Postgres
	// repository also satisfies HeroSettingsRepository (which is the case for
	// PostgresStoreSettingsRepository). Split to keep the constructor signature
	// backward-compatible.
	return &StoreSettingsService{repo: repo, storage: storage, bucket: bucket}
}

// WithHeroRepo wires the hero-settings repository into the service. Call this
// immediately after NewStoreSettingsService in modules.go.
func (s *StoreSettingsService) WithHeroRepo(heroRepo HeroSettingsRepository) *StoreSettingsService {
	s.heroRepo = heroRepo
	return s
}

// WithEditorialBannerRepo wires the editorial-banner repository into the service.
func (s *StoreSettingsService) WithEditorialBannerRepo(bannerRepo EditorialBannerRepository) *StoreSettingsService {
	s.bannerRepo = bannerRepo
	return s
}

func (s *StoreSettingsService) GetSettings(ctx context.Context) (*domain.StoreSettings, error) {
	return s.repo.Get(ctx)
}

func (s *StoreSettingsService) UpdateSettings(ctx context.Context, input UpdateStoreSettingsInput) (*domain.StoreSettings, error) {
	settings, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	if input.StoreName != nil {
		settings.StoreName = *input.StoreName
	}
	if input.LegalEntityName != nil {
		settings.LegalEntityName = input.LegalEntityName
	}
	if input.Locale != nil {
		settings.Locale = *input.Locale
	}
	if input.Currency != nil {
		settings.Currency = *input.Currency
	}
	if input.ContactEmail != nil {
		settings.ContactEmail = input.ContactEmail
	}
	if input.ContactPhone != nil {
		settings.ContactPhone = input.ContactPhone
	}
	if input.CompanyDescription != nil {
		settings.CompanyDescription = input.CompanyDescription
	}
	if input.FacebookURL != nil {
		settings.FacebookURL = input.FacebookURL
	}
	if input.InstagramURL != nil {
		settings.InstagramURL = input.InstagramURL
	}

	return s.repo.Update(ctx, *settings)
}

func (s *StoreSettingsService) UploadLogo(ctx context.Context, filename, contentType string, content io.Reader) (*domain.StoreSettings, error) {
	settings, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.storage.EnsureBucket(ctx, s.bucket); err != nil {
		return nil, err
	}

	objectKey := fmt.Sprintf("store-settings/logo/%s-%s", uuid.NewString(), filename)
	sizeBytes, err := s.storage.Upload(ctx, s.bucket, objectKey, contentType, content)
	if err != nil {
		return nil, err
	}

	bucket := s.bucket
	settings.LogoBucket = &bucket
	settings.LogoObjectKey = &objectKey
	settings.LogoContentType = &contentType
	settings.LogoSizeBytes = &sizeBytes
	return s.repo.Update(ctx, *settings)
}

func (s *StoreSettingsService) OpenLogo(ctx context.Context) (io.ReadCloser, string, error) {
	settings, err := s.repo.Get(ctx)
	if err != nil {
		return nil, "", err
	}
	if !settings.HasLogo() {
		return nil, "", domain.ErrLogoNotFound
	}
	return s.storage.Open(ctx, *settings.LogoBucket, *settings.LogoObjectKey)
}

func (s *StoreSettingsService) DeleteLogo(ctx context.Context) (*domain.StoreSettings, error) {
	settings, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if settings.HasLogo() {
		_ = s.storage.Delete(ctx, *settings.LogoBucket, *settings.LogoObjectKey)
	}
	settings.LogoBucket = nil
	settings.LogoObjectKey = nil
	settings.LogoContentType = nil
	settings.LogoSizeBytes = nil
	return s.repo.Update(ctx, *settings)
}

func (s *StoreSettingsService) GetHeroSettings(ctx context.Context) (domain.HeroSettings, error) {
	return s.heroRepo.GetHeroSettings(ctx)
}

func (s *StoreSettingsService) SaveHeroSettings(ctx context.Context, settings domain.HeroSettings) (domain.HeroSettings, error) {
	return s.heroRepo.SaveHeroSettings(ctx, settings)
}

func (s *StoreSettingsService) UploadHeroBackground(ctx context.Context, filename, contentType string, content io.Reader) (domain.HeroSettings, error) {
	if err := s.storage.EnsureBucket(ctx, s.bucket); err != nil {
		return domain.HeroSettings{}, err
	}
	objectKey := fmt.Sprintf("hero-settings/background/%s-%s", uuid.NewString(), filename)
	sizeBytes, err := s.storage.Upload(ctx, s.bucket, objectKey, contentType, content)
	if err != nil {
		return domain.HeroSettings{}, err
	}
	current, err := s.heroRepo.GetHeroSettings(ctx)
	if err != nil {
		return domain.HeroSettings{}, err
	}
	if current.HasBackground() {
		_ = s.storage.Delete(ctx, *current.BackgroundBucket, *current.BackgroundObjectKey)
	}
	bucket := s.bucket
	current.BackgroundBucket = &bucket
	current.BackgroundObjectKey = &objectKey
	current.BackgroundContentType = &contentType
	current.BackgroundSizeBytes = &sizeBytes
	return s.heroRepo.SaveHeroSettings(ctx, current)
}

func (s *StoreSettingsService) OpenHeroBackground(ctx context.Context) (io.ReadCloser, string, error) {
	settings, err := s.heroRepo.GetHeroSettings(ctx)
	if err != nil {
		return nil, "", err
	}
	if !settings.HasBackground() {
		return nil, "", domain.ErrHeroBackgroundNotFound
	}
	return s.storage.Open(ctx, *settings.BackgroundBucket, *settings.BackgroundObjectKey)
}

func (s *StoreSettingsService) DeleteHeroBackground(ctx context.Context) (domain.HeroSettings, error) {
	settings, err := s.heroRepo.GetHeroSettings(ctx)
	if err != nil {
		return domain.HeroSettings{}, err
	}
	if settings.HasBackground() {
		_ = s.storage.Delete(ctx, *settings.BackgroundBucket, *settings.BackgroundObjectKey)
	}
	settings.BackgroundBucket = nil
	settings.BackgroundObjectKey = nil
	settings.BackgroundContentType = nil
	settings.BackgroundSizeBytes = nil
	return s.heroRepo.SaveHeroSettings(ctx, settings)
}

func (s *StoreSettingsService) GetEditorialBanner(ctx context.Context) (domain.EditorialBanner, error) {
	return s.bannerRepo.GetEditorialBanner(ctx)
}

func (s *StoreSettingsService) SaveEditorialBanner(ctx context.Context, banner domain.EditorialBanner) (domain.EditorialBanner, error) {
	return s.bannerRepo.SaveEditorialBanner(ctx, banner)
}

func (s *StoreSettingsService) UploadEditorialBannerImage(ctx context.Context, filename, contentType string, content io.Reader) (domain.EditorialBanner, error) {
	if err := s.storage.EnsureBucket(ctx, s.bucket); err != nil {
		return domain.EditorialBanner{}, err
	}
	objectKey := fmt.Sprintf("editorial-banner/image/%s-%s", uuid.NewString(), filename)
	sizeBytes, err := s.storage.Upload(ctx, s.bucket, objectKey, contentType, content)
	if err != nil {
		return domain.EditorialBanner{}, err
	}
	current, err := s.bannerRepo.GetEditorialBanner(ctx)
	if err != nil {
		return domain.EditorialBanner{}, err
	}
	if current.HasImage() {
		_ = s.storage.Delete(ctx, *current.ImageBucket, *current.ImageObjectKey)
	}
	bucket := s.bucket
	current.ImageBucket = &bucket
	current.ImageObjectKey = &objectKey
	current.ImageContentType = &contentType
	current.ImageSizeBytes = &sizeBytes
	return s.bannerRepo.SaveEditorialBanner(ctx, current)
}

func (s *StoreSettingsService) OpenEditorialBannerImage(ctx context.Context) (io.ReadCloser, string, error) {
	banner, err := s.bannerRepo.GetEditorialBanner(ctx)
	if err != nil {
		return nil, "", err
	}
	if !banner.HasImage() {
		return nil, "", domain.ErrEditorialBannerImageNotFound
	}
	return s.storage.Open(ctx, *banner.ImageBucket, *banner.ImageObjectKey)
}

func (s *StoreSettingsService) DeleteEditorialBannerImage(ctx context.Context) (domain.EditorialBanner, error) {
	banner, err := s.bannerRepo.GetEditorialBanner(ctx)
	if err != nil {
		return domain.EditorialBanner{}, err
	}
	if banner.HasImage() {
		_ = s.storage.Delete(ctx, *banner.ImageBucket, *banner.ImageObjectKey)
	}
	banner.ImageBucket = nil
	banner.ImageObjectKey = nil
	banner.ImageContentType = nil
	banner.ImageSizeBytes = nil
	return s.bannerRepo.SaveEditorialBanner(ctx, banner)
}

// WithHomeSectionsRepo wires the home-sections repository into the service.
func (s *StoreSettingsService) WithHomeSectionsRepo(repo HomeSectionsRepository) *StoreSettingsService {
	s.homeSectionsRepo = repo
	return s
}

func (s *StoreSettingsService) ListHomeSections(ctx context.Context) ([]domain.HomeSection, error) {
	return s.homeSectionsRepo.ListHomeSections(ctx)
}

func (s *StoreSettingsService) SaveHomeSection(ctx context.Context, section domain.HomeSection) (domain.HomeSection, error) {
	return s.homeSectionsRepo.SaveHomeSection(ctx, section)
}

func (s *StoreSettingsService) GetSectionProductIDs(ctx context.Context, sectionID string) ([]uuid.UUID, error) {
	return s.homeSectionsRepo.GetSectionProductIDs(ctx, sectionID)
}

func (s *StoreSettingsService) SetSectionProducts(ctx context.Context, sectionID string, productIDs []uuid.UUID) error {
	return s.homeSectionsRepo.SetSectionProducts(ctx, sectionID, productIDs)
}

func (s *StoreSettingsService) GetSectionCategoryGroups(ctx context.Context, sectionID string) ([]domain.SectionCategoryGroup, error) {
	return s.homeSectionsRepo.GetSectionCategoryGroups(ctx, sectionID)
}

// SetSectionCategoryGroups replaces the curated categories (and their picked
// products) for a section. Callers cap the number of groups; the repository
// writes them transactionally.
func (s *StoreSettingsService) SetSectionCategoryGroups(ctx context.Context, sectionID string, groups []domain.SectionCategoryGroup) error {
	return s.homeSectionsRepo.SetSectionCategoryGroups(ctx, sectionID, groups)
}
