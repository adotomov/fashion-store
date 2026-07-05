package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/promotions/domain"
)

type PromotionRepository interface {
	ListPromotions(ctx context.Context) ([]domain.Promotion, error)
	GetPromotion(ctx context.Context, id uuid.UUID) (*domain.Promotion, error)
	CreatePromotion(ctx context.Context, p domain.Promotion) (*domain.Promotion, error)
	UpdatePromotion(ctx context.Context, p domain.Promotion) (*domain.Promotion, error)
	DeletePromotion(ctx context.Context, id uuid.UUID) error

	SetPromotionCategories(ctx context.Context, promotionID uuid.UUID, categoryIDs []uuid.UUID) error
	SetPromotionProductTypes(ctx context.Context, promotionID uuid.UUID, typeIDs []uuid.UUID) error
	SetPromotionProducts(ctx context.Context, promotionID uuid.UUID, productIDs []uuid.UUID) error

	// GetEffectivePrices returns the best active promotion for each of the given
	// product IDs, joining through categories and product-types as needed.
	// Only products that have at least one matching active promotion are included.
	GetEffectivePrices(ctx context.Context, productIDs []uuid.UUID) (map[uuid.UUID]domain.Promotion, error)

	// GetCategoriesWithActivePromotions returns which of the given category IDs
	// have at least one currently active promotion targeting them directly, or
	// all of them if any active "all"-target promotion exists.
	GetCategoriesWithActivePromotions(ctx context.Context, categoryIDs []uuid.UUID) ([]uuid.UUID, error)

	ListCodes(ctx context.Context) ([]domain.DiscountCode, error)
	GetCode(ctx context.Context, id uuid.UUID) (*domain.DiscountCode, error)
	FindCode(ctx context.Context, code string) (*domain.DiscountCode, error)
	CreateCode(ctx context.Context, c domain.DiscountCode) (*domain.DiscountCode, error)
	UpdateCode(ctx context.Context, c domain.DiscountCode) (*domain.DiscountCode, error)
	DeleteCode(ctx context.Context, id uuid.UUID) error
	IncrementCodeUse(ctx context.Context, id uuid.UUID) error
}
