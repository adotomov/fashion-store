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

	Status string
	Total  money.Money
	Items  []CreateOrderItemInput
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
	Items          []OrderResultItem
}

// OrderGateway persists the placed order via the orders module.
type OrderGateway interface {
	CreateOrder(ctx context.Context, input CreateOrderInput) (OrderResult, error)

	// SetShipmentInfo writes back the carrier/tracking details once a
	// shipment has been created for the order — best-effort, called after
	// CreateOrder, never blocks the checkout response on its own.
	SetShipmentInfo(ctx context.Context, orderID uuid.UUID, carrier, trackingNumber, shipmentID, status string) error
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

type CardInput struct {
	Number   string
	ExpMonth int
	ExpYear  int
	CVV      string
}

type ChargeInput struct {
	Amount   money.Money
	OrderRef string
	Card     CardInput
}

type ChargeResult struct {
	Succeeded         bool
	ProviderReference string
	FailureReason     string
}

// PaymentGateway is shaped after the Revolut Merchant API (an order/charge
// reference plus a succeeded/failed outcome) so the mock implementation
// can be swapped for a real Revolut client once the merchant account is
// verified, without touching PlaceOrder.
type PaymentGateway interface {
	Charge(ctx context.Context, input ChargeInput) (ChargeResult, error)
}
