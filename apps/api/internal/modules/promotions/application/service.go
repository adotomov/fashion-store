package application

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/promotions/domain"
)

type Service struct {
	repo PromotionRepository
}

func NewService(repo PromotionRepository) *Service {
	return &Service{repo: repo}
}

// --- Promotions CRUD ---

func (s *Service) ListPromotions(ctx context.Context) ([]domain.Promotion, error) {
	return s.repo.ListPromotions(ctx)
}

func (s *Service) GetPromotion(ctx context.Context, id uuid.UUID) (*domain.Promotion, error) {
	return s.repo.GetPromotion(ctx, id)
}

func (s *Service) CreatePromotion(ctx context.Context, input CreatePromotionInput) (*domain.Promotion, error) {
	if !input.Type.Valid() {
		return nil, domain.ErrPromotionNotFound
	}
	if !input.TargetType.Valid() {
		input.TargetType = domain.TargetAll
	}
	if input.MinQuantity < 1 {
		input.MinQuantity = 1
	}

	p := domain.Promotion{
		Name:               input.Name,
		Description:        input.Description,
		Type:               input.Type,
		ValuePercent:       input.ValuePercent,
		ValueFixedMinor:    input.ValueFixedMinor,
		ValueFixedCurrency: input.ValueFixedCurrency,
		BuyQty:             input.BuyQty,
		GetQty:             input.GetQty,
		GetDiscountPct:     input.GetDiscountPct,
		MinQuantity:        input.MinQuantity,
		TargetType:         input.TargetType,
		StartsAt:           input.StartsAt,
		EndsAt:             input.EndsAt,
		IsActive:           input.IsActive,
		Priority:           input.Priority,
	}

	created, err := s.repo.CreatePromotion(ctx, p)
	if err != nil {
		return nil, err
	}

	if err := s.setTargets(ctx, created.ID, input.TargetType, input.CategoryIDs, input.TypeIDs, input.ProductIDs); err != nil {
		return nil, err
	}
	created.CategoryIDs = input.CategoryIDs
	created.TypeIDs = input.TypeIDs
	created.ProductIDs = input.ProductIDs
	return created, nil
}

func (s *Service) UpdatePromotion(ctx context.Context, id uuid.UUID, input UpdatePromotionInput) (*domain.Promotion, error) {
	p, err := s.repo.GetPromotion(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		p.Name = *input.Name
	}
	if input.Description != nil {
		p.Description = *input.Description
	}
	if input.Type != nil {
		p.Type = *input.Type
	}
	if input.ClearValuePercent {
		p.ValuePercent = nil
	} else if input.ValuePercent != nil {
		p.ValuePercent = input.ValuePercent
	}
	if input.ClearFixed {
		p.ValueFixedMinor = nil
		p.ValueFixedCurrency = nil
	} else {
		if input.ValueFixedMinor != nil {
			p.ValueFixedMinor = input.ValueFixedMinor
		}
		if input.ValueFixedCurrency != nil {
			p.ValueFixedCurrency = input.ValueFixedCurrency
		}
	}
	if input.ClearBxgy {
		p.BuyQty = nil
		p.GetQty = nil
		p.GetDiscountPct = nil
	} else {
		if input.BuyQty != nil {
			p.BuyQty = input.BuyQty
		}
		if input.GetQty != nil {
			p.GetQty = input.GetQty
		}
		if input.GetDiscountPct != nil {
			p.GetDiscountPct = input.GetDiscountPct
		}
	}
	if input.MinQuantity != nil {
		p.MinQuantity = *input.MinQuantity
	}
	if input.TargetType != nil {
		p.TargetType = *input.TargetType
	}
	if input.ClearStarts {
		p.StartsAt = nil
	} else if input.StartsAt != nil {
		p.StartsAt = input.StartsAt
	}
	if input.ClearEnds {
		p.EndsAt = nil
	} else if input.EndsAt != nil {
		p.EndsAt = input.EndsAt
	}
	if input.IsActive != nil {
		p.IsActive = *input.IsActive
	}
	if input.Priority != nil {
		p.Priority = *input.Priority
	}

	updated, err := s.repo.UpdatePromotion(ctx, *p)
	if err != nil {
		return nil, err
	}

	if input.CategoryIDs != nil {
		if err := s.repo.SetPromotionCategories(ctx, id, *input.CategoryIDs); err != nil {
			return nil, err
		}
		updated.CategoryIDs = *input.CategoryIDs
	} else {
		updated.CategoryIDs = p.CategoryIDs
	}
	if input.TypeIDs != nil {
		if err := s.repo.SetPromotionProductTypes(ctx, id, *input.TypeIDs); err != nil {
			return nil, err
		}
		updated.TypeIDs = *input.TypeIDs
	} else {
		updated.TypeIDs = p.TypeIDs
	}
	if input.ProductIDs != nil {
		if err := s.repo.SetPromotionProducts(ctx, id, *input.ProductIDs); err != nil {
			return nil, err
		}
		updated.ProductIDs = *input.ProductIDs
	} else {
		updated.ProductIDs = p.ProductIDs
	}

	return updated, nil
}

func (s *Service) DeletePromotion(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePromotion(ctx, id)
}

// GetEffectivePrices returns the best active promotion per product.
// Only products with a matching promotion appear in the result map.
func (s *Service) GetEffectivePrices(ctx context.Context, productIDs []uuid.UUID) (map[uuid.UUID]domain.Promotion, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}
	return s.repo.GetEffectivePrices(ctx, productIDs)
}

func (s *Service) GetCategoriesWithActivePromotions(ctx context.Context, categoryIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	ids, err := s.repo.GetCategoriesWithActivePromotions(ctx, categoryIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		result[id] = true
	}
	return result, nil
}

// --- Discount Codes CRUD ---

func (s *Service) ListCodes(ctx context.Context) ([]domain.DiscountCode, error) {
	return s.repo.ListCodes(ctx)
}

func (s *Service) GetCode(ctx context.Context, id uuid.UUID) (*domain.DiscountCode, error) {
	return s.repo.GetCode(ctx, id)
}

func (s *Service) CreateCode(ctx context.Context, input CreateCodeInput) (*domain.DiscountCode, error) {
	c := domain.DiscountCode{
		Code:         strings.ToUpper(strings.TrimSpace(input.Code)),
		ValuePercent: input.ValuePercent,
		StartsAt:     input.StartsAt,
		ExpiresAt:    input.ExpiresAt,
		MaxUses:      input.MaxUses,
		IsActive:     input.IsActive,
	}
	return s.repo.CreateCode(ctx, c)
}

func (s *Service) UpdateCode(ctx context.Context, id uuid.UUID, input UpdateCodeInput) (*domain.DiscountCode, error) {
	c, err := s.repo.GetCode(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.ValuePercent != nil {
		c.ValuePercent = *input.ValuePercent
	}
	if input.ClearStarts {
		c.StartsAt = nil
	} else if input.StartsAt != nil {
		c.StartsAt = input.StartsAt
	}
	if input.ClearExpiry {
		c.ExpiresAt = nil
	} else if input.ExpiresAt != nil {
		c.ExpiresAt = input.ExpiresAt
	}
	if input.ClearMaxUses {
		c.MaxUses = nil
	} else if input.MaxUses != nil {
		c.MaxUses = input.MaxUses
	}
	if input.IsActive != nil {
		c.IsActive = *input.IsActive
	}
	return s.repo.UpdateCode(ctx, *c)
}

func (s *Service) DeleteCode(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCode(ctx, id)
}

// ValidateCode checks a discount code for validity and returns its details so
// checkout can apply the discount. Returns ErrCodeInvalid if expired/inactive,
// ErrCodeExhausted if max_uses reached, ErrCodeNotFound if it doesn't exist.
func (s *Service) ValidateCode(ctx context.Context, code string) (*domain.DiscountCode, error) {
	dc, err := s.repo.FindCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return nil, err
	}
	if !dc.IsCurrentlyValid() {
		if dc.MaxUses != nil && dc.UseCount >= *dc.MaxUses {
			return nil, domain.ErrCodeExhausted
		}
		return nil, domain.ErrCodeInvalid
	}
	return dc, nil
}

// UseCode increments the use counter after a successful order placement.
func (s *Service) UseCode(ctx context.Context, id uuid.UUID) error {
	return s.repo.IncrementCodeUse(ctx, id)
}

// setTargets replaces all target join rows for a promotion.
func (s *Service) setTargets(ctx context.Context, id uuid.UUID, targetType domain.TargetType, categoryIDs, typeIDs, productIDs []uuid.UUID) error {
	if err := s.repo.SetPromotionCategories(ctx, id, categoryIDs); err != nil {
		return err
	}
	if err := s.repo.SetPromotionProductTypes(ctx, id, typeIDs); err != nil {
		return err
	}
	return s.repo.SetPromotionProducts(ctx, id, productIDs)
}
