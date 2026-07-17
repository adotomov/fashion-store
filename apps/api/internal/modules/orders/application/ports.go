package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/orders/domain"
)

// InvoiceGateway triggers invoice generation for COD/EasyBox orders that
// reach "delivered" status — same hexagonal pattern as FulfillmentGateway.
type InvoiceGateway interface {
	GenerateForOrder(ctx context.Context, orderID uuid.UUID) error
}

// Repository persists orders and their line items. ListByUser/Create serve
// the customer-facing order history and the checkout flow; the AdminX
// methods serve the admin orders dashboard.
type Repository interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Order, error)
	CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
	Create(ctx context.Context, order domain.Order) (*domain.Order, error)

	AdminList(ctx context.Context, filter AdminListOrdersFilter) ([]domain.Order, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	FindByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, error)
	UpdateFulfillment(ctx context.Context, id uuid.UUID, input UpdateFulfillmentInput) (*domain.Order, error)
	MarkViewed(ctx context.Context, id uuid.UUID) error
	CountUnviewed(ctx context.Context) (int, error)
	Stats(ctx context.Context, since time.Time) (OrderStats, error)

	// ListAwaitingTracking returns orders with a carrier-assigned tracking
	// number that aren't delivered or cancelled yet — the fulfillment
	// module's background poller works through this list.
	ListAwaitingTracking(ctx context.Context) ([]domain.Order, error)

	// FindByProviderOrderID resolves the order behind a Revolut order id — the
	// key a payment webhook carries. Returns ErrOrderNotFound if none matches.
	FindByProviderOrderID(ctx context.Context, providerOrderID string) (*domain.Order, error)

	// MarkPaid transitions a pending_payment order to paid and records the
	// captured amount + settled payment reference. Idempotent: a no-op if the
	// order isn't pending_payment.
	MarkPaid(ctx context.Context, orderID uuid.UUID, providerReference string, capturedMinor int64) error

	// MarkPaymentFailed transitions a pending_payment order to payment_failed.
	MarkPaymentFailed(ctx context.Context, orderID uuid.UUID, reason string) error

	// GetOrderPaymentContext returns the payment/refund state used to authorize
	// and size a refund.
	GetOrderPaymentContext(ctx context.Context, orderID uuid.UUID) (OrderPaymentContext, error)

	// RecordRefund persists a refund and (when completed) advances the payment's
	// refunded total and the order's rolled-up status.
	RecordRefund(ctx context.Context, input RecordRefundInput) error

	// ListPendingPaymentOlderThan returns card orders in pending_payment created
	// before the cutoff, paired with their Revolut order id.
	ListPendingPaymentOlderThan(ctx context.Context, cutoff time.Time) ([]PendingPaymentRef, error)

	// ListPaymentTransactions returns an order's append-only payment audit trail.
	ListPaymentTransactions(ctx context.Context, orderID uuid.UUID) ([]domain.PaymentTransaction, error)
}
