package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
)

type StoreAddressService struct {
	addresses StoreAddressRepository
	settings  StoreSettingsRepository
}

func NewStoreAddressService(addresses StoreAddressRepository, settings StoreSettingsRepository) *StoreAddressService {
	return &StoreAddressService{addresses: addresses, settings: settings}
}

func (s *StoreAddressService) List(ctx context.Context) ([]domain.StoreAddress, error) {
	settings, err := s.settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	return s.addresses.List(ctx, settings.ID)
}

func (s *StoreAddressService) Create(ctx context.Context, input UpsertStoreAddressInput) (*domain.StoreAddress, error) {
	settings, err := s.settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	if input.IsDefault {
		if err := s.addresses.ClearDefault(ctx, settings.ID); err != nil {
			return nil, err
		}
	}
	return s.addresses.Create(ctx, domain.StoreAddress{
		StoreSettingsID: settings.ID,
		Label:           input.Label,
		Line1:           input.Line1,
		Line2:           input.Line2,
		City:            input.City,
		Region:          input.Region,
		PostalCode:      input.PostalCode,
		Country:         input.Country,
		IsDefault:       input.IsDefault,
	})
}

func (s *StoreAddressService) Update(ctx context.Context, id uuid.UUID, input UpsertStoreAddressInput) (*domain.StoreAddress, error) {
	existing, err := s.addresses.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.IsDefault && !existing.IsDefault {
		if err := s.addresses.ClearDefault(ctx, existing.StoreSettingsID); err != nil {
			return nil, err
		}
	}
	existing.Label = input.Label
	existing.Line1 = input.Line1
	existing.Line2 = input.Line2
	existing.City = input.City
	existing.Region = input.Region
	existing.PostalCode = input.PostalCode
	existing.Country = input.Country
	existing.IsDefault = input.IsDefault
	return s.addresses.Update(ctx, *existing)
}

func (s *StoreAddressService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.addresses.Delete(ctx, id)
}
