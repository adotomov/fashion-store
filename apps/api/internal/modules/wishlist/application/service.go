package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/wishlist/domain"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]domain.Item, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Add(ctx context.Context, userID, productID uuid.UUID) (*domain.Item, error) {
	return s.repo.Add(ctx, userID, productID)
}

func (s *Service) Remove(ctx context.Context, userID, productID uuid.UUID) error {
	return s.repo.Remove(ctx, userID, productID)
}
