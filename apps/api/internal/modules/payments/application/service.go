package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/payments/domain"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListPaymentMethods(ctx context.Context, userID uuid.UUID) ([]domain.PaymentMethod, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *Service) AddPaymentMethod(ctx context.Context, userID uuid.UUID, input CreatePaymentMethodInput) (*domain.PaymentMethod, error) {
	method := domain.PaymentMethod{
		UserID:    userID,
		Brand:     input.Brand,
		Last4:     input.Last4,
		ExpMonth:  input.ExpMonth,
		ExpYear:   input.ExpYear,
		IsDefault: input.IsDefault,
	}
	if err := method.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, method)
}

func (s *Service) UpdatePaymentMethod(ctx context.Context, userID, id uuid.UUID, input UpdatePaymentMethodInput) (*domain.PaymentMethod, error) {
	method, err := s.repo.Find(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if input.Brand != nil {
		method.Brand = *input.Brand
	}
	if input.Last4 != nil {
		method.Last4 = *input.Last4
	}
	if input.ExpMonth != nil {
		method.ExpMonth = *input.ExpMonth
	}
	if input.ExpYear != nil {
		method.ExpYear = *input.ExpYear
	}
	if input.IsDefault != nil {
		method.IsDefault = *input.IsDefault
	}
	if err := method.Validate(); err != nil {
		return nil, err
	}

	return s.repo.Update(ctx, *method)
}

func (s *Service) DeletePaymentMethod(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.Delete(ctx, userID, id)
}
