package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/orders/domain"
)

// Repository persists orders and their line items. ListByUser/Create serve
// the customer-facing order history and the checkout flow; the AdminX
// methods serve the admin orders dashboard.
type Repository interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Order, error)
	CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
	Create(ctx context.Context, order domain.Order) (*domain.Order, error)

	AdminList(ctx context.Context, filter AdminListOrdersFilter) ([]domain.Order, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	UpdateFulfillment(ctx context.Context, id uuid.UUID, input UpdateFulfillmentInput) (*domain.Order, error)
	MarkViewed(ctx context.Context, id uuid.UUID) error
	CountUnviewed(ctx context.Context) (int, error)
	Stats(ctx context.Context, since time.Time) (OrderStats, error)

	// ListAwaitingTracking returns orders with a carrier-assigned tracking
	// number that aren't delivered or cancelled yet — the fulfillment
	// module's background poller works through this list.
	ListAwaitingTracking(ctx context.Context) ([]domain.Order, error)
}
