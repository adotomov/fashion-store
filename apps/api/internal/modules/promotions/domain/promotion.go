package domain

import (
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type PromotionType string

const (
	PromotionTypePercentage PromotionType = "percentage"
	PromotionTypeFixed      PromotionType = "fixed"
	PromotionTypeBuyXGetY   PromotionType = "bxgy"
)

func (t PromotionType) Valid() bool {
	switch t {
	case PromotionTypePercentage, PromotionTypeFixed, PromotionTypeBuyXGetY:
		return true
	}
	return false
}

type TargetType string

const (
	TargetAll         TargetType = "all"
	TargetCategory    TargetType = "category"
	TargetProductType TargetType = "product_type"
	TargetProduct     TargetType = "product"
)

func (t TargetType) Valid() bool {
	switch t {
	case TargetAll, TargetCategory, TargetProductType, TargetProduct:
		return true
	}
	return false
}

type Promotion struct {
	ID          uuid.UUID
	Name        string
	Description string
	Type        PromotionType

	// Percentage: 1–100
	ValuePercent *int

	// Fixed amount off
	ValueFixedMinor    *int64
	ValueFixedCurrency *string

	// Buy X Get Y
	BuyQty         *int
	GetQty         *int
	GetDiscountPct *int // 100=free, 50=50% off

	// Minimum items in cart (for percentage/fixed too)
	MinQuantity int

	TargetType  TargetType
	CategoryIDs []uuid.UUID
	TypeIDs     []uuid.UUID
	ProductIDs  []uuid.UUID

	StartsAt *time.Time
	EndsAt   *time.Time
	IsActive bool
	Priority int

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p Promotion) IsCurrentlyActive() bool {
	if !p.IsActive {
		return false
	}
	now := time.Now()
	if p.StartsAt != nil && now.Before(*p.StartsAt) {
		return false
	}
	if p.EndsAt != nil && now.After(*p.EndsAt) {
		return false
	}
	return true
}

// ComputeDiscountedPrice returns the discounted price for a unit price under
// this promotion. For BuyXGetY the label indicates conditional applicability;
// the storefront shows the discount potential even without a full cart context.
func (p Promotion) ComputeDiscountedPrice(base money.Money) *money.Money {
	switch p.Type {
	case PromotionTypePercentage:
		if p.ValuePercent == nil {
			return nil
		}
		discounted := base.AmountMinor - (base.AmountMinor * int64(*p.ValuePercent) / 100)
		return &money.Money{AmountMinor: discounted, Currency: base.Currency}

	case PromotionTypeFixed:
		if p.ValueFixedMinor == nil {
			return nil
		}
		discounted := base.AmountMinor - *p.ValueFixedMinor
		if discounted < 0 {
			discounted = 0
		}
		return &money.Money{AmountMinor: discounted, Currency: base.Currency}

	case PromotionTypeBuyXGetY:
		// For listing pages, represent as a percentage off one unit when buying BuyQty.
		// e.g. Buy 3 Get 1 Free on 3 items = 25% off per unit.
		if p.BuyQty == nil || p.GetQty == nil || p.GetDiscountPct == nil {
			return nil
		}
		totalUnits := *p.BuyQty + *p.GetQty
		savings := base.AmountMinor * int64(*p.GetQty) * int64(*p.GetDiscountPct) / 100
		discounted := (base.AmountMinor*int64(totalUnits)-savings) / int64(totalUnits)
		return &money.Money{AmountMinor: discounted, Currency: base.Currency}
	}
	return nil
}

// Label returns a short human-readable description of the discount.
func (p Promotion) Label() string {
	switch p.Type {
	case PromotionTypePercentage:
		if p.ValuePercent != nil {
			return formatPercent(*p.ValuePercent) + " off"
		}
	case PromotionTypeFixed:
		return "Fixed discount"
	case PromotionTypeBuyXGetY:
		if p.BuyQty != nil && p.GetQty != nil && p.GetDiscountPct != nil {
			if *p.GetDiscountPct == 100 {
				return formatInt(*p.BuyQty+*p.GetQty) + " for " + formatInt(*p.BuyQty)
			}
			return "Buy " + formatInt(*p.BuyQty) + " get " + formatInt(*p.GetQty) + " at " + formatPercent(*p.GetDiscountPct) + " off"
		}
	}
	return ""
}

func formatPercent(n int) string {
	return "-" + formatInt(n) + "%"
}

func formatInt(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	result := make([]byte, 0, 4)
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}

type DiscountCode struct {
	ID           uuid.UUID
	Code         string
	ValuePercent int
	StartsAt     *time.Time
	ExpiresAt    *time.Time
	MaxUses      *int
	UseCount     int
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (d DiscountCode) IsCurrentlyValid() bool {
	if !d.IsActive {
		return false
	}
	now := time.Now()
	if d.StartsAt != nil && now.Before(*d.StartsAt) {
		return false
	}
	if d.ExpiresAt != nil && now.After(*d.ExpiresAt) {
		return false
	}
	if d.MaxUses != nil && d.UseCount >= *d.MaxUses {
		return false
	}
	return true
}

// ApplyToSubtotal computes the discount amount for the given subtotal.
func (d DiscountCode) ApplyToSubtotal(subtotal money.Money) money.Money {
	discountMinor := subtotal.AmountMinor * int64(d.ValuePercent) / 100
	return money.Money{AmountMinor: discountMinor, Currency: subtotal.Currency}
}

// EffectivePrice is the promotion-applied price for a product in the storefront.
type EffectivePrice struct {
	Price money.Money
	Label string
}
