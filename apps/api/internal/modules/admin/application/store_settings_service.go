package application

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
)

type StoreSettingsService struct {
	repo    StoreSettingsRepository
	storage MediaStorage
	bucket  string
}

func NewStoreSettingsService(repo StoreSettingsRepository, storage MediaStorage, bucket string) *StoreSettingsService {
	return &StoreSettingsService{repo: repo, storage: storage, bucket: bucket}
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
