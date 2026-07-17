package domain

import (
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type OrderStatus string

const (
	OrderStatusPending           OrderStatus = "pending"
	OrderStatusPendingPayment    OrderStatus = "pending_payment"
	OrderStatusPaid              OrderStatus = "paid"
	OrderStatusPaymentFailed     OrderStatus = "payment_failed"
	OrderStatusShipped           OrderStatus = "shipped"
	OrderStatusDelivered         OrderStatus = "delivered"
	OrderStatusCancelled         OrderStatus = "cancelled"
	OrderStatusRefunded          OrderStatus = "refunded"
	OrderStatusPartiallyRefunded OrderStatus = "partially_refunded"
)

func (s OrderStatus) Valid() bool {
	switch s {
	case OrderStatusPending, OrderStatusPendingPayment, OrderStatusPaid, OrderStatusPaymentFailed,
		OrderStatusShipped, OrderStatusDelivered, OrderStatusCancelled,
		OrderStatusRefunded, OrderStatusPartiallyRefunded:
		return true
	default:
		return false
	}
}

const (
	DeliveryMethodSpeedy  = "speedy"
	DeliveryMethodEasyBox = "easybox"
)

const (
	PaymentMethodCashOnDelivery = "cash_on_delivery"
	PaymentMethodCardOnEasyBox  = "card_on_easybox"
	PaymentMethodCardOnline     = "card_online"
)

// OrderAddress snapshots a shipping or billing address as it was at order
// time, independent of the customer's saved address book (which can change
// after the order is placed).
type OrderAddress struct {
	RecipientName string
	Phone         string
	Line1         string
	Line2         string
	City          string
	Region        string
	PostalCode    string
	CountryCode   string
}

// OrderPayment records the state of a Revolut card payment for a card_online
// order. Never present for cash_on_delivery or card_on_easybox orders, since
// those are settled at delivery time. ProviderOrderID is the Revolut order id
// (the webhook lookup key); ProviderReference is the settled payment id.
// Status moves pending → succeeded (or failed); Captured/Refunded track the
// running money totals for refunds.
type OrderPayment struct {
	ID                uuid.UUID
	OrderID           uuid.UUID
	Provider          string
	ProviderOrderID   string
	ProviderReference string
	Status            string // pending | authorised | succeeded | failed | cancelled | refunded | partially_refunded
	Amount            money.Money
	CapturedMinor     int64
	RefundedMinor     int64
	CreatedAt         time.Time
}

// Payment transaction types for the append-only payment_transactions ledger.
const (
	PaymentTxnInitiated = "initiated" // a Revolut order was opened for the customer
	PaymentTxnCaptured  = "captured"  // payment completed and funds captured
	PaymentTxnFailed    = "failed"    // payment failed, cancelled, or abandoned
	PaymentTxnRefunded  = "refunded"  // a (partial or full) refund was issued
)

// PaymentTransaction is one immutable entry in the payment audit ledger. Each
// row records a single payment-lifecycle event and the money involved at that
// moment; the sequence of a given order's rows is its full payment history.
type PaymentTransaction struct {
	ID                uuid.UUID
	OrderID           uuid.UUID
	Provider          string
	ProviderOrderID   string
	ProviderReference string
	Type              string
	Status            string
	Amount            money.Money
	CreatedAt         time.Time
}

// OrderItem snapshots a line item as it was at order time — product name,
// variant label, and price are copied rather than referencing live catalog
// rows, since those can change or be deleted after the order is placed.
type OrderItem struct {
	ID           uuid.UUID
	OrderID      uuid.UUID
	ProductID    *uuid.UUID
	ProductName  string
	VariantLabel string
	Quantity     int
	UnitPrice    money.Money
	CreatedAt    time.Time
}

type Order struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	OrderNumber string
	Status      OrderStatus
	Total       money.Money
	PlacedAt    time.Time

	ContactName  string
	ContactEmail string
	ContactPhone string

	ShippingAddress OrderAddress
	BillingAddress  OrderAddress

	DeliveryMethod string
	DeliveryFee    money.Money
	PaymentMethod  string
	Payment        *OrderPayment

	Carrier          *string
	TrackingNumber   *string
	ShipmentStatus   *string
	SpeedyShipmentID *string
	DeliveryOfficeID *string
	ViewedByAdminAt  *time.Time
	ReservationID    *uuid.UUID

	DiscountCode   *string
	DiscountAmount *money.Money

	Items []OrderItem

	CreatedAt time.Time
	UpdatedAt time.Time
}
