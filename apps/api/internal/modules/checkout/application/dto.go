package application

import (
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

// PlaceOrderResult is what PlaceOrder returns: exactly one of Order (a fully
// placed pay-on-delivery order) or PaymentRequired (an online-card order
// awaiting widget payment) is set.
type PlaceOrderResult struct {
	Order           *OrderResult
	PaymentRequired *PaymentInitiation
}

// PaymentInitiation carries what the storefront needs to mount the Revolut
// widget for an online-card order that isn't paid yet.
type PaymentInitiation struct {
	OrderID           uuid.UUID
	OrderNumber       string
	RevolutOrderID    string
	RevolutOrderToken string
	Amount            money.Money
	PaymentMethod     string
	Status            string
}

type ContactInput struct {
	FullName string
	Email    string
	Phone    string
}

type AddressInput struct {
	RecipientName string
	Phone         string
	Line1         string
	Line2         string
	City          string
	Region        string
	PostalCode    string
	CountryCode   string
}

func (a AddressInput) toDomain() domain.Address {
	return domain.Address{
		RecipientName: a.RecipientName, Phone: a.Phone, Line1: a.Line1, Line2: a.Line2,
		City: a.City, Region: a.Region, PostalCode: a.PostalCode, CountryCode: a.CountryCode,
	}
}

// PlaceOrderInput is the full checkout submission: who's ordering, where
// to ship/bill, how to deliver, and how to pay. Contact is required only
// for guest checkout (no authenticated principal) — the HTTP layer fills
// it from the signed-in profile otherwise.
type PlaceOrderInput struct {
	Contact          ContactInput
	ShippingAddress  AddressInput
	BillingAddress   AddressInput
	DeliveryMethod   string
	DeliveryOfficeID string
	PaymentMethod    string
	DiscountCode     string
}
