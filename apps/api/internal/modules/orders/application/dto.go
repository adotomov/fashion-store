package application

import (
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/orders/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type CreateOrderItemInput struct {
	ProductID    *uuid.UUID
	ProductName  string
	VariantLabel string
	Quantity     int
	UnitPrice    money.Money
}

type CreateOrderPaymentInput struct {
	Provider          string
	ProviderOrderID   string
	ProviderReference string
	Status            string
	Amount            money.Money
}

// PendingPaymentRef ties a still-unpaid card order to its Revolut order id,
// for the abandoned-payment sweeper.
type PendingPaymentRef struct {
	OrderID         uuid.UUID
	ProviderOrderID string
}

// OrderPaymentContext is the minimal payment/refund state a caller needs to
// decide whether (and how much) an order can be refunded.
type OrderPaymentContext struct {
	Status          string // the order's status
	ProviderOrderID string
	CapturedMinor   int64
	RefundedMinor   int64
	Currency        string
}

// RecordRefundInput persists one refund against an order. When State is
// "completed" the repository also advances the payment's refunded total and,
// if OrderStatus is set, the order's rolled-up status (refunded /
// partially_refunded) — all in one transaction.
type RecordRefundInput struct {
	OrderID          uuid.UUID
	ProviderRefundID string
	AmountMinor      int64
	Currency         string
	Reason           string
	State            string // pending | completed | failed
	CreatedBy        *uuid.UUID
	OrderStatus      string // new order status when the refund is completed; "" leaves it unchanged
}

type CreateOrderInput struct {
	OrderNumber string
	Status      string
	Total       money.Money
	PlacedAt    time.Time
	Items       []CreateOrderItemInput

	ContactName  string
	ContactEmail string
	ContactPhone string

	ShippingAddress domain.OrderAddress
	BillingAddress  domain.OrderAddress

	DeliveryMethod   string
	DeliveryFee      money.Money
	PaymentMethod    string
	Payment          *CreateOrderPaymentInput
	DeliveryOfficeID *string

	ReservationID *uuid.UUID

	DiscountCode   *string
	DiscountAmount *money.Money
}

// UpdateFulfillmentInput is the admin-facing mutation for an order's
// post-placement lifecycle — order status plus the mocked shipment/tracking
// fields a real logistics integration would populate via webhook. All
// fields are optional; only non-nil ones are changed.
type UpdateFulfillmentInput struct {
	Status         *string
	Carrier        *string
	TrackingNumber *string
	ShipmentStatus *string
	ShipmentID     *string
}

// AdminListOrdersFilter narrows the admin order list. UnviewedOnly is used
// to power the "unread" admin notification badge.
type AdminListOrdersFilter struct {
	Status       *string
	UnviewedOnly bool
}

// CountBreakdown is a generic (label, count) pair used for grouping orders
// by status, city, country, or delivery method on the admin dashboard.
type CountBreakdown struct {
	Label string
	Count int
}

// DailyOrderCount is one point in the admin dashboard's daily orders chart.
type DailyOrderCount struct {
	Date    time.Time
	Count   int
	Revenue money.Money
}

// OrderStats aggregates order data over a date range for the admin
// dashboard. Revenue/AvgOrderValue are zero-valued (empty currency) when
// OrderCount is 0, since there's nothing to average.
type OrderStats struct {
	OrderCount       int
	Revenue          money.Money
	AvgOrderValue    money.Money
	StatusBreakdown  []CountBreakdown
	ByCity           []CountBreakdown
	ByCountry        []CountBreakdown
	ByDeliveryMethod []CountBreakdown
	DailyCounts      []DailyOrderCount
}
