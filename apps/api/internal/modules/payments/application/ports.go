package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/payments/domain"
)

// Repository persists payment methods. Create/Update enforce that at most
// one method per user has IsDefault=true — when a new default is set, any
// existing default is cleared in the same transaction.
type Repository interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.PaymentMethod, error)
	Find(ctx context.Context, userID, id uuid.UUID) (*domain.PaymentMethod, error)
	Create(ctx context.Context, method domain.PaymentMethod) (*domain.PaymentMethod, error)
	Update(ctx context.Context, method domain.PaymentMethod) (*domain.PaymentMethod, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
}
