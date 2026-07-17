package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

// CartOwner mirrors the cart module's own type — duplicated here rather
// than imported so checkout doesn't reach into another module's domain;
// the HTTP layer translates between the two when building the gateway
// call.
type CartOwner struct {
	UserID     *uuid.UUID
	GuestToken *uuid.UUID
}

type CartLine struct {
	VariantID         uuid.UUID
	ProductID         uuid.UUID
	ProductName       string
	VariantLabel      string
	Quantity          int
	UnitPrice         money.Money
	AvailableQuantity int
}

type CartSnapshot struct {
	ID    uuid.UUID
	Lines []CartLine
}

// CartGateway lets checkout read and clear the customer's cart without
// depending on the cart module's domain/application packages directly.
type CartGateway interface {
	GetCart(ctx context.Context, owner CartOwner) (CartSnapshot, error)
	ClearCart(ctx context.Context, owner CartOwner) error
}

type ReserveLine struct {
	VariantID uuid.UUID
	Quantity  int
}

// InventoryGateway holds and finalizes stock for the items being ordered.
type InventoryGateway interface {
	Reserve(ctx context.Context, lines []ReserveLine, createdBy *uuid.UUID) (uuid.UUID, error)
	Commit(ctx context.Context, reservationID uuid.UUID, createdBy *uuid.UUID) error
	Release(ctx context.Context, reservationID uuid.UUID, createdBy *uuid.UUID) error
}

// UserGateway provisions a (possibly brand new, passwordless) user record
// for guest checkout — the same find-or-create-by-email path the auth
// module uses to provision accounts on first Google login, so a guest who
// later signs in with the same email picks up their order history.
type UserGateway interface {
	EnsureUser(ctx context.Context, email, fullName string) (uuid.UUID, error)
}

type OrderAddress struct {
	RecipientName, Phone, Line1, Line2, City, Region, PostalCode, CountryCode string
}

type OrderPaymentRecord struct {
	Provider          string
	ProviderOrderID   string
	ProviderReference string
	Status            string
	Amount            money.Money
}

type CreateOrderInput struct {
	UserID      uuid.UUID
	OrderNumber string

	ContactName  string
	ContactEmail string
	ContactPhone string

	ShippingAddress OrderAddress
	BillingAddress  OrderAddress

	DeliveryMethod   string
	DeliveryFee      money.Money
	DeliveryOfficeID string
	PaymentMethod    string
	Payment          *OrderPaymentRecord

	ReservationID uuid.UUID

	Status         string
	Total          money.Money
	DiscountCode   *string
	DiscountAmount *money.Money
	Items          []CreateOrderItemInput
}

type CreateOrderItemInput struct {
	ProductID    uuid.UUID
	ProductName  string
	VariantLabel string
	Quantity     int
	UnitPrice    money.Money
}

type OrderResultItem struct {
	ProductName  string
	VariantLabel string
	Quantity     int
	UnitPrice    money.Money
}

// OrderResult is what PlaceOrder returns — a checkout-owned read model so
// the HTTP layer never needs to import the orders module's domain types.
type OrderResult struct {
	ID             uuid.UUID
	OrderNumber    string
	Status         string
	Total          money.Money
	DeliveryMethod string
	DeliveryFee    money.Money
	PaymentMethod  string
	PlacedAt       time.Time
	DiscountCode   *string
	DiscountAmount *money.Money
	Items          []OrderResultItem
}

// OrderGateway persists and settles the placed order via the orders module.
type OrderGateway interface {
	CreateOrder(ctx context.Context, input CreateOrderInput) (OrderResult, error)

	// SetShipmentInfo writes back the carrier/tracking details once a
	// shipment has been created for the order — best-effort, called after
	// CreateOrder, never blocks the checkout response on its own.
	SetShipmentInfo(ctx context.Context, orderID uuid.UUID, carrier, trackingNumber, shipmentID, status string) error

	// FindByProviderOrderID loads the finalize snapshot for the order behind a
	// Revolut order id (the webhook lookup key). Returns ErrOrderNotFound if
	// none matches.
	FindByProviderOrderID(ctx context.Context, providerOrderID string) (OrderForFinalize, error)

	// MarkPaid settles a pending_payment order once its payment is confirmed.
	MarkPaid(ctx context.Context, orderID uuid.UUID, providerReference string, capturedMinor int64) error

	// MarkPaymentFailed moves a pending_payment order to payment_failed.
	MarkPaymentFailed(ctx context.Context, orderID uuid.UUID, reason string) error

	// GetPaymentContext returns the payment/refund state used to authorize and
	// size a refund.
	GetPaymentContext(ctx context.Context, orderID uuid.UUID) (OrderPaymentContext, error)

	// RecordRefund persists a refund and advances the order's refund state.
	RecordRefund(ctx context.Context, input RecordRefundInput) error

	// ListPendingPaymentOlderThan returns card orders still awaiting payment
	// since before the cutoff — the abandoned-payment sweeper's work list.
	ListPendingPaymentOlderThan(ctx context.Context, cutoff time.Time) ([]PendingPaymentRef, error)

	// GetStatusByNumber returns just an order's status, keyed by its order
	// number — powers the storefront's post-payment status poll for guests.
	GetStatusByNumber(ctx context.Context, orderNumber string) (string, error)
}

// PendingPaymentRef ties an order to its Revolut order id for the sweeper.
type PendingPaymentRef struct {
	OrderID         uuid.UUID
	ProviderOrderID string
}

// OrderForFinalize is the subset of a placed order the settlement path needs
// to commit stock and book the shipment from a webhook, without reloading the
// customer's original in-memory checkout submission.
type OrderForFinalize struct {
	ID               uuid.UUID
	OrderNumber      string
	Status           string
	UserID           uuid.UUID
	ReservationID    *uuid.UUID
	DeliveryMethod   string
	DeliveryOfficeID string
	PaymentMethod    string
	ContactName      string
	ContactEmail     string
	ContactPhone     string
	ShippingAddress  OrderAddress
	Total            money.Money
}

// OrderPaymentContext mirrors the orders module's payment/refund state.
type OrderPaymentContext struct {
	OrderStatus     string
	ProviderOrderID string
	CapturedMinor   int64
	RefundedMinor   int64
	Currency        string
}

// RecordRefundInput asks the orders module to persist one refund and, when
// completed, advance the order's rolled-up status.
type RecordRefundInput struct {
	OrderID          uuid.UUID
	ProviderRefundID string
	Amount           money.Money
	Reason           string
	State            string
	CreatedBy        *uuid.UUID
	OrderStatus      string
}

// CreateShipmentInput is what checkout hands the fulfillment module after an
// order is created.
type CreateShipmentInput struct {
	Provider       string
	DeliveryMethod string
	OfficeID       string

	ContactName string
	Phone       string
	Email       string
	Address     OrderAddress

	RequireCOD bool
	CODAmount  money.Money
	Ref1       string
}

type ShipmentResult struct {
	ShipmentID string
	ParcelID   string
}

// FulfillmentGateway lets checkout gate delivery methods on whether their
// provider is enabled and create the real shipment once an order is placed
// — all without checkout importing the fulfillment module's domain or
// repository directly. The EasyBox locker dropdown talks to fulfillment's
// own public offices endpoint directly, so no office-search method is
// needed here.
type FulfillmentGateway interface {
	IsProviderEnabled(ctx context.Context, provider string) bool
	CreateShipment(ctx context.Context, input CreateShipmentInput) (ShipmentResult, error)
}

// Revolut order states (lowercased by the gateway). We only branch on
// "completed"; the rest are logged/ignored until a later webhook resolves them.
const (
	PaymentStateCompleted = "completed"
	PaymentStateCancelled = "cancelled"
	PaymentStateFailed    = "failed"
)

// CreatePaymentOrderInput asks the gateway to open a Revolut order the
// customer will pay via the embedded widget. Card data never passes through
// here — the widget tokenizes it client-side.
type CreatePaymentOrderInput struct {
	Amount        money.Money
	OrderNumber   string // sent as merchant_order_ext_ref
	CustomerEmail string
}

// PaymentOrder is the created/queried Revolut order. Token is the public token
// the frontend widget mounts; ID is the server-side order id used for webhook
// lookups and refunds.
type PaymentOrder struct {
	ID          string
	Token       string
	State       string
	AmountMinor int64
	Currency    string
}

type RefundInput struct {
	ProviderOrderID string
	Amount          money.Money
	Reason          string
}

type RefundResult struct {
	ID    string
	State string
}

// PaymentGateway is the Revolut Merchant seam: open an order for the widget,
// re-fetch it authoritatively when a webhook arrives, and refund it. The mock
// implementation stands in until the merchant account is live, without
// PlaceOrder/FinalizePaidOrder needing to change.
type PaymentGateway interface {
	CreateOrder(ctx context.Context, input CreatePaymentOrderInput) (PaymentOrder, error)
	GetOrder(ctx context.Context, providerOrderID string) (PaymentOrder, error)
	Refund(ctx context.Context, input RefundInput) (RefundResult, error)
}

// DiscountInfo carries the result of a valid discount code lookup.
type DiscountInfo struct {
	CodeID       uuid.UUID
	ValuePercent int
}

// DiscountGateway validates discount codes and records their use after a
// successful order — isolated from the promotions module's domain so
// checkout never imports promotions packages directly.
type DiscountGateway interface {
	ValidateCode(ctx context.Context, code string) (DiscountInfo, error)
	UseCode(ctx context.Context, codeID uuid.UUID) error
}

// InvoiceGateway triggers invoice generation without importing the invoicing
// module's domain directly — same hexagonal pattern as FulfillmentGateway.
type InvoiceGateway interface {
	GenerateForOrder(ctx context.Context, orderID uuid.UUID) error
}
