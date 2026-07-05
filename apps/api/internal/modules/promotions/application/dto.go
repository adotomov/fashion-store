package application

import (
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/promotions/domain"
)

type CreatePromotionInput struct {
	Name        string
	Description string
	Type        domain.PromotionType

	ValuePercent       *int
	ValueFixedMinor    *int64
	ValueFixedCurrency *string

	BuyQty         *int
	GetQty         *int
	GetDiscountPct *int

	MinQuantity int
	TargetType  domain.TargetType
	CategoryIDs []uuid.UUID
	TypeIDs     []uuid.UUID
	ProductIDs  []uuid.UUID

	StartsAt *time.Time
	EndsAt   *time.Time
	IsActive bool
	Priority int
}

type UpdatePromotionInput struct {
	Name        *string
	Description *string
	Type        *domain.PromotionType

	ValuePercent       *int
	ClearValuePercent  bool
	ValueFixedMinor    *int64
	ValueFixedCurrency *string
	ClearFixed         bool

	BuyQty         *int
	GetQty         *int
	GetDiscountPct *int
	ClearBxgy      bool

	MinQuantity *int
	TargetType  *domain.TargetType
	CategoryIDs *[]uuid.UUID
	TypeIDs     *[]uuid.UUID
	ProductIDs  *[]uuid.UUID

	StartsAt     *time.Time
	ClearStarts  bool
	EndsAt       *time.Time
	ClearEnds    bool
	IsActive     *bool
	Priority     *int
}

type CreateCodeInput struct {
	Code         string
	ValuePercent int
	StartsAt     *time.Time
	ExpiresAt    *time.Time
	MaxUses      *int
	IsActive     bool
}

type UpdateCodeInput struct {
	ValuePercent *int
	StartsAt     *time.Time
	ClearStarts  bool
	ExpiresAt    *time.Time
	ClearExpiry  bool
	MaxUses      *int
	ClearMaxUses bool
	IsActive     *bool
}

// CodeValidation is returned by ValidateCode so checkout can apply the discount.
type CodeValidation struct {
	CodeID       uuid.UUID
	ValuePercent int
}
