package domain

import (
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

func (s OrderStatus) Valid() bool {
	switch s {
	case OrderStatusPending, OrderStatusPaid, OrderStatusShipped, OrderStatusDelivered, OrderStatusCancelled:
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

// OrderPayment records the outcome of a (mocked, Revolut-shaped) charge
// attempt for card_online orders. Never present for cash_on_delivery or
// card_on_easybox orders, since those are settled at delivery time.
type OrderPayment struct {
	ID                uuid.UUID
	OrderID           uuid.UUID
	Provider          string
	ProviderReference string
	Status            string // succeeded | failed
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

	Items []OrderItem

	CreatedAt time.Time
	UpdatedAt time.Time
}
